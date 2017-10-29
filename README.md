# DIY robot bartender

Builds with Raspberry Pi and Go. 
How it works:

<a href="https://www.youtube.com/watch?v=8zgrqq7ezRE
" target="_blank"><img src="http://img.youtube.com/vi/8zgrqq7ezRE/0.jpg" 
alt="DIY robot bartender" width="240" height="180" border="10" /></a>

About Nalivator (russian) - https://habrahabr.ru/post/327220/

Speech synthesis - https://speechkit.yandex.com/dev


# Scheme

Here is a principial scheme of pump connection.

![Scheme](https://4te.me/img/scheme.png)

* 3,3V - GPIO pin
* 12V - additional power for pump ([PD-45](https://github.com/fote/nalivator9000/blob/master/docs/psu.pdf))
* R1=150 Ohm, R2=300 Ohm
* Transistor - [BDX33B](http://www.farnell.com/datasheets/56743.pdf)

# How to build

1. Install Golang
2. Install govendor:
```go get -u github.com/kardianos/govendor```
3. Get the source:
```go get github.com/fote/nalivator9000```
4. Now compile:
```cd $GOPATH/src/github.com/fote/nalivator9000 && go build -v```
