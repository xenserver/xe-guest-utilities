package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"strings"
	xenstoreclient "xe-guest-utilities/xenstoreclient"
)

func die(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func usage() {
	die(
		`Usage: xenstore read key [ key ... ]
                list key [ key ... ]
                write key value [ key value ... ]
                rm key [ key ... ]
                exists key [ key ... ]
                ls [ key ... ]
                chmod key mode [modes...]
                watch [-n NR] key [ key ... ]`)
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

		for _, subPath := range result {
			fmt.Println(subPath)
		}
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

var max_width = 80

const TAG = " = \"...\""
const XENSTORE_ABS_PATH_MAX = 3072
const STRING_MAX = (XENSTORE_ABS_PATH_MAX + 1024)

func sanitise_value(val string) string {
	var builder strings.Builder

	for _, r := range val {
		switch {
		case r >= ' ' && r <= '~' && r != '\\':
			builder.WriteRune(r)
		case r == '\t':
			builder.WriteString("\\t")
		case r == '\n':
			builder.WriteString("\\n")
		case r == '\r':
			builder.WriteString("\\r")
		case r == '\\':
			builder.WriteString("\\\\")
		case r < '\010':
			builder.WriteString(fmt.Sprintf("%03o", r))
		default:
			builder.WriteString(fmt.Sprintf("x%02x", r))
		}
	}

	return builder.String()
}

func do_xs_ls(xs xenstoreclient.XenStoreClient, path string, depth int) {
	result, err := xs.List(path)
	if err != nil {
		die("xs_ls error: %v %s", err, path)
	}
	for _, sub_path := range result {
		if len(sub_path) == 0 {
			continue
		}
		slash := "/"
		if len(path) > 0 && path[len(path)-1] == '/' {
			slash = ""
		}
		newPath := path + slash + sub_path

		col := 0
		for col < depth {
			fmt.Print(" ")
			col++
		}

		n := len(sub_path)
		if n > (max_width - len(TAG) - col) {
			n = (max_width - len(TAG) - col)
		}
		fmt.Printf(sub_path[:n])
		col += n

		if len(newPath) >= STRING_MAX {
			fmt.Println(":")
		} else {
			val, err := xs.Read(newPath)
			if err != nil {
				fmt.Println(":")
			} else {
				val = sanitise_value(val)
				if (col + len(val) + len(TAG)) > max_width {
					n := max_width - col - len(TAG)
					if n < 0 {
						n = 0
					}
					fmt.Printf(" = \"%s...\"\n", val[:n])
				} else {
					fmt.Printf(" = \"%s\"\n", val)
				}
			}
		}

		do_xs_ls(xs, newPath, depth+1)
	}
}

func xs_ls(script_name string, args []string) {
	if len(args) == 1 && args[0] == "-h" {
		die("Usage: %s [ key ... ]", script_name)
	}

	const TIOCGWINSZ = 0x5413
	winsize, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), TIOCGWINSZ)
	if err == nil {
		max_width = int(winsize.Col)
	}

	xs := new_xs()
	if len(args) != 0 {
		for _, key := range args[:] {
			do_xs_ls(xs, key, 0)
		}
	} else {
		domain_id, err := xs.Read("domid")
		if err == nil {
			domain_path, err := xs.GetDomainPath(strings.TrimRight(domain_id, "\x00"))
			if err == nil {
				do_xs_ls(xs, strings.TrimRight(domain_path, "\x00"), 0)
			}
		}
	}
}

func xs_chmod(script_name string, args []string) {
	if len(args) < 2 || args[0] == "-h" {
		die("Usage: %s key mode [modes...]", script_name)
	}

	var err error
	key := args[0]
	var perms []xenstoreclient.Permission

	for _, m := range args[1:] {
		if len(m) > 1 {
			var p xenstoreclient.Permission
			switch m[0] {
			case 'n':
				p.Pe = xenstoreclient.PERM_NONE
			case 'r':
				p.Pe = xenstoreclient.PERM_READ
			case 'w':
				p.Pe = xenstoreclient.PERM_WRITE
			case 'b':
				p.Pe = xenstoreclient.PERM_READWRITE
			default:
				err = errors.New("Invalid mode string")
			}
			if err == nil {
				var id uint64
				id, err = strconv.ParseUint(m[1:], 10, 0)
				if err == nil {
					p.Id = uint(id)
					perms = append(perms, p)
				}
			}
		} else {
			err = errors.New("Invalid mode length")
		}
		if err != nil {
			die("%s error: %v", script_name, err)
		}
	}

	xs := new_xs()
	err = xs.SetPermission(key, perms)
	if err != nil {
		die("%s error: %v", script_name, err)
	}
}

func xs_watch_die(script_name string) {
	die("Usage: %s [-n NR] key [ key ... ]", script_name)
}

func xs_watch(script_name string, args []string) {
	if len(args) == 0 || args[0] == "-h" {
		xs_watch_die(script_name)
	}

	nr, index := 0, 0
	if strings.HasPrefix(args[0], "-n") {
		if len(args[0]) > 2 {
			n, err := strconv.Atoi(args[0][2:])
			if err != nil || n < 1 || len(args) == 1 {
				xs_watch_die(script_name)
			}
			nr = n
			index = 1
		} else if len(args) > 2 {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				xs_watch_die(script_name)
			}
			nr = n
			index = 2
		} else {
			xs_watch_die(script_name)
		}
	}

	xs := new_xs()
	if out, err := xs.Watch(args[index:]); err == nil {
		for i := 0; nr == 0 || i < nr; i++ {
			if e, ok := <-out; ok {
				fmt.Println(e.Path)
			} else {
				os.Exit(1)
			}
		}
		xs.StopWatch()
	} else {
		os.Exit(1)
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
	case "ls":
		xs_ls(script_name, args)
	case "chmod":
		xs_chmod(script_name, args)
	case "watch":
		xs_watch(script_name, args)
	default:
		usage()
	}
}
