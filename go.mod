module tserver

go 1.16

replace proto/pb => /home/twl/gocode/src/proto/pb

require (
	github.com/gin-gonic/gin v1.7.2
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.39.0
	proto/pb v1.0.0
)
