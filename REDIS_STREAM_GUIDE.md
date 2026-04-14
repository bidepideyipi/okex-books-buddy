select 1
# 查看 Stream 中所有消费者组
xinfo groups signals
# 取5条消息
xrange signals - + count 5

# 清空 Stream 中所有消息，但保留 Stream 键和消费者组
XTRIM signals MAXLEN 0

# 测试消息
XADD signals "*" timestamp "1775740000000" inst_id "SOL-USDT-SWAP" bar "15m" prediction "1" price "120.5" line1 "0.015" line2 "0.040"
