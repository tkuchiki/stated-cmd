# stated-cmd

## Usage

```console
$ ./stated-cmd --help
usage: stated-cmd [<flags>] <command> [<args> ...]

stated command runner

Flags:
  --help     Show context-sensitive help (also try --help-long and --help-man).
  --version  Show application version.

Commands:
  help [<command>...]
    Show help.

  run --cmd=CMD [<flags>]
    run command

  fail --file=FILE
    failed list

$ ./stated-cmd run --help
usage: stated-cmd run --cmd=CMD [<flags>]

run command

Flags:
      --help           Show context-sensitive help (also try --help-long and --help-man).
      --version        Show application version.
      --cmd=CMD        command
  -c, --concurrency=1  concurrency

$ ./stated-cmd fail --help
usage: stated-cmd fail --file=FILE

failed list

Flags:
      --help       Show context-sensitive help (also try --help-long and --help-man).
      --version    Show application version.
  -f, --file=FILE  state file

```

## Examples

```console
# running
# md5(sleep) -> c9fab33e9458412c527c3fe8a13ee37d
$ seq 1 3 | ./stated-cmd run --cmd "sleep"
create .c9fab33e9458412c527c3fe8a13ee37d.conf
2018/08/02 14:13:56 sleep 3
2018/08/02 14:13:59 sleep 1
2018/08/02 14:14:00 sleep 2

# not running
$ seq 1 3 | ./stated-cmd run --cmd "sleep"
load .c9fab33e9458412c527c3fe8a13ee37d.conf
```

```console
$ seq 1 3 | time ./stated-cmd run --cmd "sleep" -c 3
create .c9fab33e9458412c527c3fe8a13ee37d.conf
2018/08/02 14:15:39 sleep 3
2018/08/02 14:15:39 sleep 1
2018/08/02 14:15:39 sleep 2
./stated-cmd run --cmd "sleep" -c 3  0.01s user 0.02s system 0% cpu 3.023 total
```

```console
$ cat .c9fab33e9458412c527c3fe8a13ee37d.conf | jq .
{
  "1": "fail",
  "2": "success",
  "3": "fail"
}

$ ./stated-cmd fail -f .c9fab33e9458412c527c3fe8a13ee37d.conf
1
3

$ ./stated-cmd fail -f .c9fab33e9458412c527c3fe8a13ee37d.conf | ./stated-cmd run --cmd sleep -c 2
load .c9fab33e9458412c527c3fe8a13ee37d.conf
2018/08/02 15:28:30 sleep 1
2018/08/02 15:28:30 sleep 3

$ cat .c9fab33e9458412c527c3fe8a13ee37d.conf | jq .
{
  "1": "success",
  "2": "success",
  "3": "success"
}
```
