
### 云函数打包工具

#### windows下打包会导致文件没有可执行权限
1. 腾讯云会 405提示容器异常退出
2. 实际是无法执行对应的命令

#### 安装
```shell
go get -u github.com/losemy/build_util
go install github.com/losemy/build_util
```

### 打包方式示例
```shell
rm main.zip # 删除打包文件
GOOS=linux GOARCH=amd64 go build -o main # 编译linux x64数据
build_util -output main.zip main scf_bootstrap config.yaml # 使用打包工具进行打包
```

