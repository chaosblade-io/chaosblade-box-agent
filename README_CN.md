# Chaosblade-exec-os: 基础资源混沌实验场景执行器
![license](https://img.shields.io/github/license/chaosblade-io/chaosblade.svg)

## 介绍
探针(Agent)主要作为平台端建联、命令下发通道和数据收集等功能，所以如果需要对目标集群或主机进行演练，需要在端侧的目标集群或主机上安装探针，以便将平台编排好的演练转化成命令，下发到目标机器上

## 使用
此项目可以单独编译后使用，但更建议通过 [chaosblade-box-agent](https://github.com/chaosblade-io/chaosblade-box-agent) 安装使用。详细的中文使用文档请参考：https://chaosblade.io/docs/getting-started/installation-and-deployment/agent-install

## 编译
此项目采用 golang 语言编写，所以需要先安装最新的 golang 版本，最低支持的版本是 1.11。Clone 工程后进入项目目录执行以下命令进行编译：
```shell script
make
```
如果在 mac 系统上，编译当前系统的版本，请执行：
```shell script
make build_darwin
```
如果想在 mac 系统上，编译 linux 系统版本，请执行：
```shell script
make build_linux
```
你也可以只 clone [chaosblade-box-agent](https://github.com/chaosblade-io/chaosblade-box-agent) 项目，在项目目录下执行 `make` 或 `make build_linux` 来统一编译。

安装agent的步骤：
```bash
./chaosctl.sh install -k 015667e5361b4b0c9d42e1c10afe1d61 -p  [应用名]  -g  [应用分组]  -P  [agent端口号]  -t [chaosblade-box ip:port]
```

安装示例:
```bash
./chaosctl.sh install -k  0813d72a71ba41ed986e507e2e0ead1b  -p  chaos-default-app  -g  chaos-default-app-group  -P 19527 -t 127.0.0.1
```

## 缺陷&建议
欢迎提交缺陷、问题、建议和新功能，所有项目（包含其他项目）的问题都可以提交到[Github Issues](https://github.com/chaosblade-io/chaosblade/issues) 

你也可以通过以下方式联系我们：
* 钉钉群（推荐）：23177705
* Gitter room: [chaosblade community](https://gitter.im/chaosblade-io/community)
* 邮箱：chaosblade.io.01@gmail.com
* Twitter: [chaosblade.io](https://twitter.com/ChaosbladeI)

## 参与贡献
我们非常欢迎每个 Issue 和 PR，即使一个标点符号，如何参加贡献请阅读 [CONTRIBUTING](CONTRIBUTING.md) 文档，或者通过上述的方式联系我们。

## 开源许可证
Chaosblade-exec-os 遵循 Apache 2.0 许可证，详细内容请阅读 [LICENSE](LICENSE)

