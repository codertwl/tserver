package main

import (
        "errors"
        "fmt"
        "golang.org/x/net/context"
        "google.golang.org/grpc"
        "net"
        "net/http"
        "sync"

        //      "google.golang.org/grpc/reflection"
        "github.com/gin-gonic/gin"
        "proto/pb"
)

type server struct{}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
        fmt.Println("replay...")
        return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func RegisterGrpc(host string) {

        ctx := context.Background()
        lis, addr, err := ListenServAddr(ctx, host)
        if err != nil {
                fmt.Println(err)
        }
        fmt.Println(addr)

        s := grpc.NewServer()                // 创建gRPC服务器
        pb.RegisterHelloServer(s, &server{}) // 在gRPC服务端注册服务

        // reflection.Register(s) //在给定的gRPC服务器上注册服务器反射服务
        // Serve方法在lis上接受传入连接，为每个连接创建一个ServerTransport和server的goroutine。
        // 该goroutine读取gRPC请求，然后调用已注册的处理程序来响应它们。
        err = s.Serve(lis)
        if err != nil {
                fmt.Printf("failed to serve: %v", err)
                return
        }
}

func Ping(ctx *gin.Context) {
        fmt.Println("ping...")
}

func RegisterGin(host string) {
        ctx := context.Background()
        lis, addr, err := ListenServAddr(ctx, host)
        if err != nil {
                fmt.Println(err)
        }
        fmt.Println(addr)

        g := gin.New()
        g.POST("/ping", Ping)
        s := http.Server{Handler: g}
        s.Serve(lis)
        if err != nil {
                fmt.Printf("failed to serve: %v", err)
                return
        }
}

func GetListenAddr(a string) (string, error) {
        addrTcp, err := net.ResolveTCPAddr("tcp", a)
        fmt.Println("===============>", addrTcp)
        if err != nil {
                return "", err
        }

        addr := addrTcp.String()
        host, _, err := net.SplitHostPort(addr)
        if err != nil {
                return "", err
        }

        if len(host) == 0 {
                return GetServAddr(addrTcp)
        }

        return addr, nil
}

func GetInterIp() (string, error) {
        addrs, err := net.InterfaceAddrs()
        if err != nil {
                return "", err
        }

        for _, address := range addrs {
                // check the address type and if it is not a loopback the display it
                if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
                        if ipnet.IP.To4() != nil {
                                //fmt.Println(ipnet.IP.String())
                                return ipnet.IP.String(), nil
                        }
                }
        }

        /*
                for _, addr := range addrs {
                        //fmt.Printf("Inter %v\n", addr)
                        ip := addr.String()
                        if "10." == ip[:3] {
                                return strings.Split(ip, "/")[0], nil
                        } else if "172." == ip[:4] {
                                return strings.Split(ip, "/")[0], nil
                        } else if "196." == ip[:4] {
                                return strings.Split(ip, "/")[0], nil
                        } else if "192." == ip[:4] {
                                return strings.Split(ip, "/")[0], nil
                        }

                }
        */

        return "", errors.New("no inter ip")
}

func GetServAddr(a net.Addr) (string, error) {
        addr := a.String()
        host, port, err := net.SplitHostPort(addr)
        if err != nil {
                return "", err
        }
        if len(host) == 0 {
                host = "0.0.0.0"
        }

        ip := net.ParseIP(host)

        if ip == nil {
                return "", fmt.Errorf("ParseIP error, host: %s", host)
        }
        /*
                fmt.Println("ADDR TYPE", ip,
                        "IsGlobalUnicast",
                        ip.IsGlobalUnicast(),
                        "IsInterfaceLocalMulticast",
                        ip.IsInterfaceLocalMulticast(),
                        "IsLinkLocalMulticast",
                        ip.IsLinkLocalMulticast(),
                        "IsLinkLocalUnicast",
                        ip.IsLinkLocalUnicast(),
                        "IsLoopback",
                        ip.IsLoopback(),
                        "IsMulticast",
                        ip.IsMulticast(),
                        "IsUnspecified",
                        ip.IsUnspecified(),
                )
        */

        raddr := addr
        if ip.IsUnspecified() {
                // 没有指定ip的情况下，使用内网地址
                inerip, err := GetInterIp()
                if err != nil {
                        return "", err
                }

                raddr = net.JoinHostPort(inerip, port)
        }

        //slog.Tracef("ServAddr --> addr:[%s] ip:[%s] host:[%s] port:[%s] raddr[%s]", addr, ip, host, port, raddr)

        return raddr, nil
}

func ListenServAddr(ctx context.Context, addr string) (net.Listener, string, error) {
        paddr, err := GetListenAddr(addr)
        if err != nil {
                return nil, "", err
        }

        tcpAddr, err := net.ResolveTCPAddr("tcp", paddr)
        if err != nil {
                return nil, "", err
        }

        netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
        if err != nil {
                return nil, "", err
        }

        laddr, err := GetServAddr(netListen.Addr())
        if err != nil {
                netListen.Close()
                return nil, "", err
        }

        return netListen, laddr, nil
}

func main() {
        var wg sync.WaitGroup
        wg.Add(2)
        go RegisterGrpc(":8972")
        go RegisterGin(":8973")
        wg.Wait()
}

