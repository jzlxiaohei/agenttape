实验 1：纯文本回答
请只回复 hello，不要使用工具

观察：

初始 prompt 由哪些部分组成
是否只有一次模型请求
UserPromptSubmit 到 Stop 的路径
实验 2：只读取一个文件
读取 README.md，总结第一段

观察：

模型第一次如何产生 tool call
PreToolUse 与网络 response 的先后关系
tool result 如何加入下一次请求
实验 3：修改一个文件

观察：

Read、Edit、Write 如何选择
写文件前是否触发权限
diff 是模型生成还是本地 harness 生成
修改结果如何回传模型
实验 4：执行失败
运行一个确定会失败的测试

观察：

PostToolUseFailure
stderr 如何编码
模型如何根据失败结果重试
harness 是否自动截断过长输出
实验 5：触发 compaction

用长会话触发上下文压缩，重点对比：

PreCompact
→ 压缩前最后一个网络请求
→ PostCompact
→ 压缩后第一个网络请求

这是理解 context management 最有价值的实验之一。

实验 6：子 Agent

观察主 Agent 和子 Agent 是否：

使用不同 system prompt
拥有不同工具集合
拥有独立 conversation history
只向主 Agent 返回摘要
共享或隔离工作目录