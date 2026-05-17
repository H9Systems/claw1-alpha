package main

// Version is set at build time via -ldflags "-X main.version=…".
// If unset, falls back to "dev".
var version = "dev"