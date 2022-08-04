package main

import "github.com/rommms07/dogfetch"

func main() {
	alaskan := dogfetch.GetByName("Alaskan Malamute")

	println(alaskan.Name)
}
