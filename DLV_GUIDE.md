dlv debug main.go
break main.go:10
c continue 运行到断点
s step 单步执行（进入函数）
n next 单步执行（不进入函数）
print 打印当前变量值

stack
goroutine 打印当前goroutine信息