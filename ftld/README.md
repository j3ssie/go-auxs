## ftld

Finding more TLD from Public Suffix

## Usage

```shell
# simple usage
echo 'target.com' | ftld -c 50

target.es
target.cn
target.com.cn

# input the org directly
ftld -c 50 -o 'target'

target.es
target.cn
www.target.com.cn

# with prefix
echo 'target.com' | ftld -c 50 -p 'www' -p 'dev'

dev.target.es
www.target.cn
```
