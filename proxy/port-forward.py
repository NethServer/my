#!/usr/bin/env python3
"""Forward TCP traffic from a listen port to a local target port."""

import socket
import sys
import threading


def forward(src, dst):
    try:
        while True:
            data = src.recv(65536)
            if not data:
                break
            dst.sendall(data)
    except Exception:
        pass
    finally:
        src.close()
        dst.close()


def handle(client, target_port):
    try:
        upstream = socket.create_connection(("127.0.0.1", target_port))
    except Exception:
        client.close()
        return
    threading.Thread(target=forward, args=(client, upstream), daemon=True).start()
    forward(upstream, client)


def main():
    target_port = int(sys.argv[1]) if len(sys.argv) > 1 else 8443
    listen_port = int(sys.argv[2]) if len(sys.argv) > 2 else 443
    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("0.0.0.0", listen_port))
    srv.listen(128)
    print(f"Forwarding port {listen_port} -> {target_port}")
    while True:
        client, _ = srv.accept()
        threading.Thread(target=handle, args=(client, target_port), daemon=True).start()


if __name__ == "__main__":
    main()
