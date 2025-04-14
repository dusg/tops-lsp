package lsp

import (
	"log"
	"os"
)

var info *log.Logger = log.New(os.Stdout, "INFO: ", log.LstdFlags)
var warn *log.Logger = log.New(os.Stderr, "WARN: ", log.LstdFlags)
var error *log.Logger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
