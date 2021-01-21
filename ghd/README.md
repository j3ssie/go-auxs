## Github dorks

```bash
$ cat dorks.txt

https://github.com/search?q=%22{{.Raw}}%22+password&type=Code
https://github.com/search?q=%22{{.Org}}%22+password&type=Code
https://github.com/search?q=%22{{.Raw}}%22+npmrc%20_auth&type=Code
https://github.com/search?q=%22{{.Org}}%22+npmrc%20_auth&type=Code

$ ghd -d dorks.txt -u tesla.com

https://github.com/search?q=%22tesla.com%22+password&type=Code
https://github.com/search?q=%22tesla%22+password&type=Code
https://github.com/search?q=%22tesla.com%22+npmrc%20_auth&type=Code
https://github.com/search?q=%22tesla%22+npmrc%20_auth&type=Code

```