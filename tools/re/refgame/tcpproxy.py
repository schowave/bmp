import socket,threading
def pipe(a,b):
    try:
        while True:
            d=a.recv(65536)
            if not d: break
            b.sendall(d)
    except Exception: pass
    finally:
        try: a.close(); b.close()
        except Exception: pass
s=socket.socket(); s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1); s.bind(('0.0.0.0',5902)); s.listen(5)
while True:
    c,_=s.accept(); u=socket.create_connection(('127.0.0.1',5901))
    threading.Thread(target=pipe,args=(c,u),daemon=True).start(); threading.Thread(target=pipe,args=(u,c),daemon=True).start()
