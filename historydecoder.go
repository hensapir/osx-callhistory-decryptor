package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// TagSz size of the tag
	TagSz = 0x10
	// NonceSz size of the nonce
	NonceSz = 0x10

	sqlStmt = `
		select ZDATE,
			case ZANSWERED
				when 1 then 'true'
				else 'false'
			end,
			case ZORIGINATED
				when 1 then 'true'
				else 'false'
			end,
			case ZCALLTYPE
				when 1 then 'CellPhone'
				else 'FaceTime'
			end,
			ZISO_COUNTRY_CODE,
			ZADDRESS
		from ZCALLRECORD
		order by ZDATE`
)

// DecipherHistory opens the database and writes CSV output to output
// returns number of rows processed or an error (if any)
func DecipherHistory(database string, key []byte, output io.Writer) (int, error) {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	rows, err := db.Query(sqlStmt)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	csvOut := csv.NewWriter(output)
	defer csvOut.Flush()
	csvOut.Write([]string{"Date", "Answered?", "Outgoing?", "Type", "Country", "Number/Address"})

	numRecords := 0

	for rows.Next() {
		var callOffset float64
		var answered string
		var originated string
		var calltype string
		var country string
		var blob = make([]byte, 255)

		err = rows.Scan(&callOffset, &answered, &originated, &calltype, &country, &blob)
		if err != nil {
			return 0, err
		}
		callTime := calcCallTime(callOffset)

		address, err := Decipher(blob, key)
		if err != nil {
			return 0, err
		}
		csvOut.Write([]string{callTime.Format("2006-01-02 15:04:05Z0700"),
			answered, originated, calltype, country, string(address)})

		numRecords++
	}

	err = rows.Err()
	if err != nil {
		return 0, err
	}
	return numRecords, nil
}

//Decipher deciphers ZADDRESS from OS X call history.
func Decipher(data, key []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, nil
	}
	tag := data[0:TagSz]
	nonce := data[TagSz : TagSz+NonceSz]
	ct := data[0x20:]

	cttag := make([]byte, TagSz+len(ct))
	copy(cttag[0:len(ct)], ct)
	copy(cttag[len(ct):], tag)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSz)
	if err != nil {
		log.Fatal(err)
	}

	pt, err := gcm.Open(nil, nonce, cttag, nil)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

//Cipher text conforming to ZADDRESS encryption pattern
func Cipher(text, key []byte) ([]byte, error) {
	if text == nil || len(text) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(key)
	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSz)
	nonce := make([]byte, NonceSz)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	cttag := gcm.Seal(nil, nonce, text, nil)
	ct := make([]byte, len(cttag)+NonceSz)
	copy(ct[0:0x10], cttag[len(cttag)-TagSz:])
	copy(ct[0x10:0x20], nonce[:])
	copy(ct[0x20:], cttag[:len(cttag)-TagSz])

	return ct, nil
}

// calcCallTime calculates the call time.
func calcCallTime(callOffset float64) time.Time {
	if callOffset < 0 {
		callOffset = 0
	}
	startDate := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return startDate.Add(time.Second * time.Duration(int64(callOffset)))
}
