#!/bin/bash
iptables -t nat -F
# 别忘了去/etc/sysctl.conf中增加 net.ipv4.ip_forward = 1
  
# 血环
iptables -t nat -A PREROUTING -p tcp --dport 13001 -j DNAT --to-destination 169.150.222.245:8090
iptables -t nat -A POSTROUTING -p tcp -d 169.150.222.245 --dport 8090 -j SNAT --to-source 你服务器的内网IP

# 翡翠梦境
iptables -t nat -A PREROUTING -p tcp --dport 13002 -j DNAT --to-destination 169.150.222.71:8090
iptables -t nat -A POSTROUTING -p tcp -d 169.150.222.71 --dport 8090 -j SNAT --to-source 你服务器的内网IP

# 霍格
iptables -t nat -A PREROUTING -p tcp --dport 13003 -j DNAT --to-destination 169.150.222.69:8090
iptables -t nat -A POSTROUTING -p tcp -d 169.150.222.69 --dport 8090 -j SNAT --to-source 你服务器的内网IP

# 拉文郡
iptables -t nat -A PREROUTING -p tcp --dport 13004 -j DNAT --to-destination 169.150.222.223:8090
iptables -t nat -A POSTROUTING -p tcp -d 169.150.222.223 --dport 8090 -j SNAT --to-source 你服务器的内网IP
