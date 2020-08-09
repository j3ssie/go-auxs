chrunk
======
Run your command against really really big file.

## Installation

```shell
go get -u github.com/j3ssie/go-auxs/chrunk

```

## Examples

```shell

chrunk -i /tmp/really_big_file.txt -s 20000

chrunk -i /tmp/really_big_file.txt -cmd 'echo "--> {}"'

chrunk -p 20 -i /tmp/really_big_file.txt -cmd 'echo "--> {}"'

cat really_big_file.txt | chrunk -p 10 -cmd 'echo "--> {}"'

```

## Usage

```
Usage of chrunk:
  -c int
    	Set the concurrency level (default 1)
  -clean
    	Clean junk file after done
  -cmd string
    	Command to run after chunked content
  -i string
    	Input file to split
  -o string
    	Output foldeer contains list of filename
  -p int
    	Number of parts to split
  -prefix string
    	Prefix output filename
  -s int
    	Number of lines to split file (default 10000)
```