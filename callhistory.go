/*
OS X Call history decryptor
Copyright (C) 2016  n0fate (GPL2 license)
Copyright (C) 2019  rusq   (golang implementation)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rusq/callhistory/historydecoder"
)

const (
	version = "1.1"
)

var (
	envKey = os.Getenv("KEY")

	strKey         = flag.String("k", envKey, "Base64 key value from OS X keychain.")
	filename       = flag.String("f", "CallHistory.storedata", "filename with call data. Get it from:\n"+os.Getenv("HOME")+"/Library/Application Support/CallHistoryDB/\n")
	outputFilename = flag.String("o", "", "output csv filename.  If not specified, result is output to stdout")
	versionOnly    = flag.Bool("v", false, "print version and quit")
)

func printHeader() {
	fmt.Fprintf(os.Stderr, "MacOS X Call History Decryptor v.%s © 2018 rusq\n"+
		"Based on Call History Decryptor © 2016 n0fate\n",
		version)
}

func main() {
	flag.Parse()

	printHeader()
	if *versionOnly {
		return
	}

	key, err := historydecoder.DecodeB64Key([]byte(*strKey))
	if err != nil {
		log.Fatalf("%s: make sure you have supplied the key via -k or KEY env variable", err)
	}

	outFile := os.Stdout
	if *outputFilename != "" || *outputFilename == "-" {
		outFile, err := os.Create(*outputFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer outFile.Close()
	}

	log.Printf("*** filename: %s\n", *filename)

	numRecords, err := historydecoder.DecipherHistory(*filename, key, outFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("*** finished. %d records processed\n", numRecords)
}
