package main

import (
	xenstoreclient "../xenstoreclient"
	"fmt"
	"os"
	"strings"
)

func die(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func usage() {
	die(
		`Usage: xenstore read key [ key ... ]
                write key value [ key value ... ]
                rm key [ key ... ]
                exists key [ key ... ]`)
}

func new_xs() xenstoreclient.XenStoreClient {
	xs, err := xenstoreclient.NewXenstore(0)
	if err != nil {
		die("xenstore.Open error: %v", err)
	}

	return xs
}

func xs_read(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" {
		die("Usage: %s key [ key ... ]", script_name)
	}

	xs := new_xs()
	for _, key := range args[:] {
		result, err := xs.Read(key)
		if err != nil {
			die("%s error: %v", script_name, err)
		}

		fmt.Println(result)
	}
}

func xs_list(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" {
		die("Usage: %s key [ key ... ]", script_name)
	}

	xs := new_xs()
	for _, key := range args[:] {
		result, err := xs.List(key)
		if err != nil {
			die("%s error: %v", script_name, err)
		}

		fmt.Println(result)
	}
}

func xs_write(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" || len(args)%2 != 0 {
		die("Usage: %s key value [ key value ... ]", script_name)
	}

	xs := new_xs()
	for i := 0; i < len(args); i += 2 {
		key := args[i]
		value := args[i+1]

		err := xs.Write(key, value)
		if err != nil {
			die("%s error: %v", script_name, err)
		}
	}
}

func xs_rm(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" {
		die("Usage: %s key [ key ... ]", script_name)
	}

	xs := new_xs()
	for _, key := range args[:] {
		err := xs.Rm(key)
		if err != nil {
			die("%s error: %v", script_name, err)
		}
	}
}

func xs_exists(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" {
		die("Usage: %s key [ key ... ]", script_name)
	}

	xs := new_xs()
	for _, key := range args[:] {
		_, err := xs.Read(key)
		if err != nil {
			die("%s error: %v", script_name, err)
		}
	}
}

func main() {
	var operation string
	var args []string

	script_name := os.Args[0]
	if strings.Contains(script_name, "-") {
		operation = script_name[strings.LastIndex(script_name, "-")+1:]
		args = os.Args[1:]
	} else {
		if len(os.Args) < 2 {
			usage()
		}
		operation = os.Args[1]
		script_name = script_name + " " + operation
		args = os.Args[2:]
	}

	switch operation {
	case "read":
		xs_read(script_name, args)
	case "list":
		xs_list(script_name, args)
	case "write":
		xs_write(script_name, args)
	case "rm":
		xs_rm(script_name, args)
	case "exists":
		xs_exists(script_name, args)
	default:
		usage()
	}
}
