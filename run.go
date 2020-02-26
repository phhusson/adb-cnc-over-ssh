package main

import (
        "bytes"
        "io"
        "fmt"
        "log"
        "time"
        "os"
        "os/exec"
        "strings"

        "github.com/gliderlabs/ssh"
)

func adbGetprop(port uint32, prop string) string{
    var stdout bytes.Buffer

    remote := fmt.Sprintf("127.0.0.1:%v", port)

    cmd := exec.Command("adb", "-s", remote, "wait-for-device", "shell", "getprop", prop)
    cmd.Stdout = &stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    if err != nil {
        fmt.Println("Failed getting prop", err)
    }

    return strings.TrimSpace(string(stdout.Bytes()))
}


func adbConnect(port uint32) {
    remote := fmt.Sprintf("127.0.0.1:%v", port)
    cmd := exec.Command("adb", "connect", remote)
    err := cmd.Start()
    if err != nil {
        fmt.Println("Failed calling adb connect")
    }
    err = cmd.Wait()
    if err != nil {
        fmt.Println("Failed calling adb connect")
    }
}

func adbCheck(port uint32) bool {
    remote := fmt.Sprintf("127.0.0.1:%v", port)
    cmd := exec.Command("adb", "-s", remote, "shell", "true")
    err := cmd.Start()
    if err != nil {
        fmt.Println("Failed calling adb shell true")
    }
    err = cmd.Wait()
    if err != nil {
        fmt.Println("Failed calling adb shell true")
        return false
    }
    fmt.Println("Managed to call adb shell true")
    return true
}

func handleConnect(port uint32) {
    // wait up to 10s for adb to appear
    for i := 0; i < 100; i++ {
        adbConnect(port)
        time.Sleep(100 * time.Millisecond)
        fmt.Println("Waiting connect to", port)
        if adbCheck(port) {
            break;
        }
    }

    product := adbGetprop(port, "ro.product.product.device")
    if product == "" {
        product = adbGetprop(port, "ro.vendor.product.device")
    }
    if product == "" {
        product = adbGetprop(port, "ro.product.device")
    }

    realSerial := adbGetprop(port, "ro.serialno")

    log.Println("Connected to ", port, " which is ", product, " with serial ", realSerial)
}

func main() {
    log.Println("starting ssh server on port 2222...")

    forwardHandler := &ssh.ForwardedTCPHandler{}

    server := ssh.Server{
        LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
            return false
        }),
        Addr: ":2222",
        Handler: ssh.Handler(func(s ssh.Session) {
            io.WriteString(s, "Remote forwarding available...\n")
            select {}
        }),
        ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
            log.Println("attempt to bind", host, port, "granted")

            doConnect := port > 10000 && host == "localhost"
            if doConnect {
                go handleConnect(port)
            }
            return doConnect
        }),
        RequestHandlers: map[string]ssh.RequestHandler{
            "tcpip-forward":        forwardHandler.HandleSSHRequest,
            "cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
        },
    }

    log.Fatal(server.ListenAndServe())
}

