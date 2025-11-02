NSM项目是Network Service Mesh项目的缩写，是一个基于Kubernetes的网络服务网格项目。

本目录中的cmd-template是一个模板，cmd-nse-firewall是基于该模板实现的一个NSE（Network Service Endpoint），用于实现防火墙功能。

为了开发出更多的NSE，我们可以需要了解NSM的相关知识，包括：

1. NSM的架构
2. NSE的工作原理
3. NSE的开发流程

需要基于cmd-template，模仿cmd-nse-firewall的开发流程，开发出更多的NSE。

为了简化开发难度，我们先从解耦cmd-nse-firewall-vpp开始，把代码分解出来保存在新的目录cmd-nse-firewall-vpp-temp中：
1. cmd-nse-firewall-vpp-temp是为了打包成一个firewall的容器，放入Kubernetes集群中NSM环境下作为NSE使用，但代码中出了firewall相关的代码，其他代码都是通用的，我需要把代码解耦出来，最好是分文件调用，把功能文件提取出来，便于后续实现新的NSE。
2. 解耦出来的功能代码需要能够进行单独的测试，不依赖于NSM环境，可以直接在本地用go测试功能是否正常
3. 解耦出来的代码需要有良好的文档说明，说明每个文件的功能和作用
4. 保持良好的代码风格，清晰的目录结构，不允许把很多文档放在文件夹根目录里
5. 解耦的代码需要与原来的代码保持一致的功能和接口，不能改变原来的代码逻辑，能够打包成一样的镜像

