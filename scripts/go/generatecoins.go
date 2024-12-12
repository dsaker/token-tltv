/*
generatetokens generates tokens to be used in token-tltv.

It will print the plaintext token once when you run the program. They will not be
saved to the json file so you must save them to some place safe after you run the
program.

Usage:

	Generatetokens [flags] [path ...]

The flags are:

	-o
	    The file where you want the tokens json file to be output. Default is /tmp/
	-n
	    The number of tokens you want to be generated. Default is 10.
*/
package main

import (
	"flag"
	"log"
	"talkliketv.click/tltv/internal/test"
	"time"
)

func main() {
	var outputPath string
	var numTokens int

	filename := "tokens-" + time.Now().Format("20060102150405") + ".json"
	flag.StringVar(&outputPath, "o", "/tmp/", "file path where you want json token file")
	flag.IntVar(&numTokens, "n", 0, "the number of tokens to generate")
	flag.Parse()

	plaintexts, err := test.CreateTokensFile(outputPath+"/"+filename, numTokens)
	if err != nil {
		log.Fatal(err)
	}

	for plain := range plaintexts {
		println(plain)
	}
}
