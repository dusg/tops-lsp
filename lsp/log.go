package lsp

import (
	"log"
	"os"
)

var ilog *log.Logger = log.New(os.Stdout, "INFO: ", log.LstdFlags)
var wlog *log.Logger = log.New(os.Stderr, "WARN: ", log.LstdFlags)
var elog *log.Logger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
