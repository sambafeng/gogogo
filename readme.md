# Dinp安装攻略
## -2、架构图

[![env ports](https://raw.githubusercontent.com/chinesejie/gogogo/master/images/arche.png)](https://github.com/chinesejie/gogogo)

a. 用户把代码打包为.tar.gz，交给Builder打包为一个Docker image

b. 拿到Builder产出的Docker image去Dashboard创建一个App，设置好实例数、内存大小、image地址，O了。Dashboard把用户填写的这些信息写入MySQL

c. Server定期从MySQL同步用户期望的数据，姑且称之为desired state

d. 部署在所有计算节点的Agent与Server之间有心跳通信，收集本机的剩余内存量和container列表，姑且称之为real state

e. Server对比desired state和real state，发现某个App的实例数少了就去调度新的计算节点创建新实例，如果发现某个App实例数多了，就干掉多余的实例

f. Server同时会分析real state，组织出路由信息写入redis

g. Router定期从redis中获取路由信息

h. Router通常部署多个，前面部署LVS，注册一个域名，比如apps.io，把*.apps.io这个泛域名解析到LVS VIP，整个流程就通了


## -1、环境配置【下面就为了实验，把所有的组件往一个机器110.95.241.31安装】

[![env ports](https://raw.githubusercontent.com/chinesejie/gogogo/master/images/ports.png)](https://github.com/chinesejie/gogogo)



## 0、dinp的golang代码准备

该文特别长，该提到的点我都会提到。先下载代码[https://github.com/chinesejie/gogogo]，

我们设置的代码的根路径是/root/gogogo。

```
cd /root/
git clone https://github.com/chinesejie/gogogo
cd gogogo
rm -rf .git #为了让子路径的.git不冲突，就删除根路径的.git
find . -depth -type d -name .git1 -exec sh -c 'mv "${0}" "${0%1*}"' {} \;
#${0%1*}表示表示从右边开始，删除第一个 1 符号及右边的字符
```

最后一句话的意义是：

> 因为gogogo下面都是github.com模块的代码，因此子路径有.git文件，我想把所有代码都传到git单库，~~而不是链接跳转~~。
#所以我用在gogogo下面使用 find . -depth -type d -name .git -exec sh -c 'mv "${0}" "${0}1"' {} \;
#把所有的.git 文件夹改成 .git1 ,所以在github.com/chinesejie/gogogo能看到很多.git1文件。                                                       #下载代码之后，执行最后一句话就是把所有的.git1改名成.git 


然后你进入到每一个github.com的子目录都能 git status或者git remote -v看看github.com的子目录库对应版本到底是什么。。

####PS#####

> 由于下文dinp/gorouter库使用了sed命令替换了 ``code.google.com/``gogoprotobuf字符串，顺便把.git/index也给替换掉了，所以git status会报错如下：                                                          [error: bad index file sha1 signature \n fatal: index file corrupt这报错没关系。可以忽略]``

## 1、安装docker1.7 版本

docker容器最早受到RHEL完善的支持是从最近的CentOS 7.0开始的，CentOS-7的docker直接包含在官方镜像源的Extras仓库，直接yum install docker即可。

需要注意的是CentOS 6.x与7.0的安装是有一点点不同的，CentOS-6上docker的安装包叫docker-io，并且来源于Fedora epel库，这个仓库维护了大量的没有包含在发行版中的软件，我们需要安装一下epel。

0）在经过了上一篇文章_<使用 yum 快速升级 CentOS 6.3 内核到 3.10.x>_的洗礼之后，
centos6.3下使用yum install docker-io，默认的docker 版本是_**`0.7.0【太低了】`**_

不过我们想使用_1.7.1_的版本。。

1）安装一下epel：

```
 rpm -ivh http://dl.fedoraproject.org/pub/epel/6/x86_64/epel-release-6-8.noarch.rpm
```

2）创建两个文件：

* epel.repo[旧的epel.repo保留一份]

```
[epel]
name=Extra Packages for Enterprise Linux 6 - $basearch
#baseurl=http://download.fedoraproject.org/pub/epel/6/$basearch
baseurl=http://mirrors.sohu.com/fedora-epel/6/$basearch
#mirrorlist=https://mirrors.fedoraproject.org/metalink?repo=epel-6&arch=$basearch
failovermethod=priority
enabled=1
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-6
  
[epel-debuginfo]
name=Extra Packages for Enterprise Linux 6 - $basearch - Debug
#baseurl=http://download.fedoraproject.org/pub/epel/6/$basearch/debug
baseurl=http://mirrors.sohu.com/fedora-epel/6/$basearch/debug
#mirrorlist=https://mirrors.fedoraproject.org/metalink?repo=epel-debug-6&arch=$basearch
failovermethod=priority
enabled=0
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-6
gpgcheck=1
  
[epel-source]
name=Extra Packages for Enterprise Linux 6 - $basearch - Source
#baseurl=http://download.fedoraproject.org/pub/epel/6/SRPMS
baseurl=http://mirrors.sohu.com/fedora-epel/6/SRPMS
#mirrorlist=https://mirrors.fedoraproject.org/metalink?repo=epel-source-6&arch=$basearch
failovermethod=priority
enabled=0
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-6
gpgcheck=1
```

* CentOS-Base.repo[旧的CentOS-Base.repo保留一份]

阿里云Linux安装软件镜像源:
CentOS 6下面生成CentOS-Base.repo：

```
wget -O /etc/yum.repos.d/CentOS-Base.repo http://mirrors.aliyun.com/repo/Centos-6.repo
```

3）执行如下三行命令：

```
yum clean all
yum makecache

###记得安装下，，，，device-mapper
yum -y install device-mapper.x86_64


# 最后是安装docker
yum -y install docker-io #安装docker-io
service docker start #启动docker服务

 
使用docker version #检查服务是否正常
```


4）疑难解答：

其他用户使用docker?
因为docker是使用root安装的，其他账户要使用，需要将账户加入docker group，命令：usermod -a -G docker $yourAcc

docker.io和docker-engine的区别？
这两者都可以，但是两者是互斥的。**_只是docker-engine是docker官方维护的，docker.io是Ubuntu 维护的。_**

## 2、安装uic【需要数据库uic】

现在给大家介绍一个小组件：User Information Center，下文以UIC代称。

> 项目简介

这个组件本来是给我们的监控系统用的，后来稍微扩展了一下，也成了我们PaaS平台的一个组件。那她到底是干啥的呢？主要有两个功能：

* 维护了User与Team的对应关系，即对人做了分组
* 提供了一个简单的sso（即单点登录）功能

简单解释一下：比如对于一个监控系统，主要功能就是收集数据，然后分析处理、报警，报警的通常形式就是发送报警短信、报警邮件了，所以监控系统需要通过某种途径拿到用户的手机号和邮箱地址。
有人说，那直接让用户把手机号、邮箱配置到监控系统里好了，我们说这不是个好方法。因为监控系统中通常会配置很多策略模板，手机号、邮箱可能会出现在不同的模板中，如果一旦修改，要改好多地方。那这个简单，我们不让用户直接配置手机号、邮箱，而是配置用户名，然后系统再根据用户名拿到联系方式，没错，的确应该这么处理。
但是，一个策略模板如果触发阈值发出报警，通常不是发给一个人，而是发给一批人，比如UIC挂了，UIC的研发工程师和运维工程师都应该收到报警，所以通常来讲，策略模板配置的报警接收人应该是一个团队名称。故而我们抽象出了一个Team的概念，其实就是一批人的集合。
冥冥中，我们感觉这个User和Team的对应关系可能不光监控系统需要，所以我们就单独做为一个模块开发了，后来发现，PaaS也需要，因为一个app通常也是由多个人来开发的，这多个人都可以对这个app发起上线、回滚、扩容等操作，所以app的管理者不应该是一个人，而应该是一个Team。
这样一来，我们只需要在一个公共的地方：UIC，维护User与Team的对应关系即可。
再来说说sso功能，UIC之所以提供sso功能，是因为监控系统、PaaS平台通常都是由多个组件协调工作的，那种需要跟用户通过页面交互的组件需要有登陆认证，这显然是sso的应用场景，所以UIC提供了一个很简单的sso机制

> 项目安装【觉得烦直接看第9个步骤】

这是一个Java的项目， 安装起来比较简单，搞过Java的人肯定一看就懂，步骤如下：
1、安装JDK，配置JAVA_HOME环境变量
从oracle官网下载或者直接安装openjdk都可以，我们用的JDK1.7，也可以用1.6，代码中没有使用1.7的特性，然后导出JAVA_HOME环境变量，比如

```
`export JAVA_HOME=/path/to/your/jdk
export PATH=$JAVA_HOME/bin:$PATH`
```

2、安装Apache Ant
嗯，这个项目没有用maven，小项目我还是喜欢ant，把$ANT_HOME/bin配置到PATH中 ant官网下载地址：http://ant.apache.org/bindownload.cgi[建议apache-ant-1.5.2版本]
下载解压缩，然后跟JDK类似，配置一下环境变量

```
`export JAVA_HOME=/path/to/your/jdk
export ANT_HOME=/path/to/your/ant
export PATH=$JAVA_HOME/bin:$ANT_HOME/bin:$PATH`
```

3、编译UIC代码

```
`git clone https://github.com/UlricQin/uic.git
cd uic
ant`
```

build.xml是ant用到的编译配置文件，里边用的JDK1.7，如果你是JDK1.6，需要把build.xml中的1.7改成1.6
4、安装web容器
我的做法是：干掉webapps目录的所有内容，修改conf/server.xml
把URIEncoding配置到Connector上，防止乱码，端口也可以根据系统占用情况灵活分配，此处使用默认的8080

```
`<Connector port="8080" protocol="HTTP/1.1" connectionTimeout="20000" redirectPort="8443" URIEncoding="UTF-8" />`
```

然后去配置文件最下面找到`<Host>`标签，在里边增加`<Context>`标签

```
`<Context path="" docBase="/path/to/uic/web" />`
```

5、修改UIC配置文件
配置文件的位置是：`/path/to/uic/web/WEB-INF/config.txt`
我们简单解释一下各个配置项的作用：

* jdbcUrl、user、password是数据库的连接配置
* 假如你的DB也打算叫uic，那么`create database uic;`，然后使用`/path/to/uic/scripts/schema.sql`初始化数据库
* memcacheAddrs是用逗号分隔的多个memcached地址
* canRegister取值true或者false，以此控制是否开放注册功能，如果配置成false，不开放注册，需要使用schema.sql中初始化的root账号登陆，然后创建普通用户账号
* token 因为UIC保存了很多用户信息，有些接口不是谁都可以访问的，用token做了一个简单限制，这是一个随机字符串，我们之后介绍的PaaS Builder和Dashboard也都需要配置相同的token才可以

6、成了，启动tomcat即可

```
cd uic-tomcat/bin
sh start.sh
```

7、memcached 安装流程
先把 memcached 安装下：
yum -y install memcached
/etc/rc.d/init.d/memcached start

8、效果
访问：http://ipxxx:8080/ 用户名root 密码abc

9、上面八个步骤太繁琐了，直接 快捷方式，直接使用我的github。看下面

```
`# 换成我的github是 https://github.com/chinesejie/gogogo下面的uic跟uic-tomcat
# 注意根路径是/home/pay/uic
cd /home/pay
git clone `https://github.com/chinesejie/`gogogo`
cd `gogogo/java`
mv uic ../..
mv uic-tomcat ../..
vi ../../`````uic/web/WEB-INF/config.txt ```````# 记得修改下`uic/web/WEB-INF/config.txt```
cd ../../uic-tomcat/bin && sh start.sh ``
```

> 二次开发

笔者才疏学浅，不免可能会有一些bug之类的，读者可以帮忙做一些bugfix或者根据自己公司的需求开发一些新feature，我简单介绍一下开发的注意事项

* 项目使用国人开发的一个叫做JFinal的框架开发完成，所以，读者要先去了解一下这个框架
* 项目最好使用eclipse for jee版本来开发，即使你习惯用idea，使用JFinal开发项目最好先使用eclipse
* src、conf、frame这三个目录都是source folder，不要仅仅把src作为source folder使用
* 开发阶段lib下面的jetty-server-8.1.8.jar和web/WEB-INF/lib下的所有jar包引入classpath
* 找到`com.ulricqin.uic.config.Config.java`，里边有main方法，启动即可测试
* 使用jetty这种方式修改了代码，JFinal会自动感知并且自动编译加载修改的类，比较方便
* 使用jetty这种方式可能会提示slf4j的某个类发生了多次重复加载，这个问题我一直没弄明白，不过不影响项目测试，线上使用tomcat没有这个问题

O了，UIC就给大家介绍这么多，之后会陆续介绍PaaS的其他组件。

## 3. Docker registry

这是Docker的一个SCM，存放Image用的，安装方式有多种，大家Google一下吧，最简单的安装方法是：

```
`docker pull registry`
```

当然了，拉取国外的image可能比较慢，大家可以找个国内的镜像用一用。
编辑/etc/sysconfig/docker，设置下面的参数：

```
other_args="-H tcp://110.95.241.31:2375 -H unix://var/run/docker.sock --insecure-registry 110.95.241.31:5000"
```

-H表示这个docker服务器既支持tcp连接又支持本地socket连接，我们在本地打命令`docker info`默认就是使用本地socket连接本地服务器。我们也可以用docker -H tcp://110.95.241.31:2375 info使用tcp连接来访问本地的服务器。
@@@@PS@@@ 

> PS：如果使用docker -H tcp://127.0.0.1:2375 是连不上的docker-server的。


--insecure-registry设置表示支持docker-server支持 非ssl的的连接通过5000端口推送image镜像到docker-server的私有仓库。当然了，此时docker-server必须启动 带有私有仓库registry的容器。

然后，再重启下docker:

```
service docker restart 
service docker status#失败了。看/var/log/docker 的日志
```

## 4. 启动registry容器

使用下面的命令来启动 registry容器。。

```
mkdir -p /root/dinp/data/registry && docker run --name registry -d -v /root/dinp/data/registry:/tmp/registry -p 5000:5000 registry
```

可以用docker ps来查看正在运行的容器。

首先，/root/dinp/data/registry是宿主机的存放image镜像的位置，而映射到registry容器上的/tmp/registry位置。
所以通过-v参数可以支持把一个宿主机上的目录挂载到镜像里。

5000代表宿主机端口，而后面的5000表示容器机器的端口。

--name registry表示指定了容器进行的名字是registry，不指定的话一般容器实例的名字都是docker-server随意生成的一串。

-d 表示后台运行

最后的registry 表示 image镜像的名字【可以通过docker images来查得】

## 5. nsenter

centso6.3系统自带了/usr/bin/nsenter工具，没有就自己装一个。。

编辑docker-enter.sh 如下：

```
#!/bin/sh

if [ -e $(dirname "$0")/nsenter ]; then
    # with boot2docker, nsenter is not in the PATH but it is in the same folder
    NSENTER=$(dirname "$0")/nsenter
else
    NSENTER=nsenter
fi

if [ -z "$1" ]; then
    echo "Usage: `basename "$0"` CONTAINER [COMMAND [ARG]...]"
    echo ""
    echo "Enters the Docker CONTAINER and executes the specified COMMAND."
    echo "If COMMAND is not specified, runs an interactive shell in CONTAINER."
else
    PID=$(docker inspect --format "{{.State.Pid}}" "$1")
    if [ -z "$PID" ]; then
        exit 1
    fi
    shift

    OPTS="--target $PID --mount --uts --ipc --net --pid --"

    if [ -z "$1" ]; then
        # No command given.
        # Use su to clear all host environment variables except for TERM,
        # initialize the environment variables HOME, SHELL, USER, LOGNAME, PATH,
        # and start a login shell.
        "$NSENTER" $OPTS su - root
    else
        # Use env to clear all host environment variables.
        "$NSENTER" $OPTS env --ignore-environment -- "$@"
    fi
fi
```

然后运行sh docker-enter.sh <containID>即可进入容器内看东西。。

## 6.  安装go

```

cd /home/pay/soft
wget https://storage.googleapis.com/golang/go1.4.1.linux-amd64.tar.gz
tar -zxvf  go1.4.1.linux-amd64.tar.gz
mv go go1.4
vi ~/.bash_profile
#增加：
export GOROOT=/home/pay/soft/go1.4
export GOPATH=/root/gogogo
export PATH=$GOROOT/bin:$GOPATH:$PATH

然后再
source ~/.bash_profile


```

## 7. 编译builder【需要数据库builder】

> 设计理念

* 平台不管编译，用户自己编译好，然后把编译好的代码扔给平台
* 不管是什么语言的代码，最终都打包成一个docker image，通过docker来做规范化

好，下面我们详细解释

> 编译与否的问题

用户的代码要部署在PaaS平台上，首先要做的就是把代码扔给平台，那么问题来了，通过什么方式扔给平台呢？通过git？还是直接上传？扔上来的是源代码？还是已经编译完成的？解释型语言当然是不需要编译的，但是编译型的呢，比如golang，java，我们要让用户给我们.go、.java文件然后我们去编译？还是用户编译好，直接给我们二进制，给我们.class文件？
平台刚起步，我们希望在可接受的范围内，做的事情越少越好，这样出错的几率就越少，之后平台慢慢发展，可以再加入一些新的feature。所以我们选择让用户把编译之后的代码交给我们，比如java的话，就扔个war包给我们即可；golang的话需要编译成二进制（编译成64位Linux的，因为平台是64位Linux）。
有些PaaS，比如tsuru，是让用户把代码push到某个指定repo的指定分支的，然后后端的git receiver就会触发编译、上线脚本。看起来是挺好的，但是这样处理比较麻烦，坑太多，以后再说吧
接下来是要确定通过什么方式把代码扔给平台，我们目前支持两种方式，一个是在页面直接上传，一个是给一个可以下载的http地址，平台去下载。

> 使用docker来做规范化打包

用户的代码可能是不同语言写的，比较常见的比如Java、PHP、Python，用户给我们的是一个tar.gz包，我们要怎么把其变成一个docker image呢？
最简单的实现方式：dinp提供一个base image，里边就只是Ubuntu或者centos的环境，用户自己搞定runtime依赖，比如Java的话，用户的tar.gz包中应该包含JDK、tomcat、webapp，然后约定一个启动脚本，比如就是根目录的control文件，平台只需要`./control start`即可启动。如果用户是PHP的代码，需要把nginx、php-fpm、code打包进来。
但是这种方式太麻烦，如果是golang的话可以使用上面的方式，因为golang是静态编译的，也没啥依赖。Java、php、Python、Ruby让用户这么搞，那不得哭了……
我们可以针对不同的语言做不同的base image，比如Java的，base image中提前把JDK和tomcat做好，用户直接上传一个war包，搞定！比如PHP的base image，提前把nginx、php-fpm之类的做好，用户直接把php code上传，搞定！
嗯，这也是Builder平台现在采用的方式，我们做了一些base image，让用户在页面上选择使用哪个，用户选择了base image，也就间接的告诉了我们他是什么类型的代码了（比如java的程序肯定要选择一个java的base image）。然后我们根据不同类型的程序生成一个Dockerfile，把用户的代码和base image揉在一起，搞定！

> 安装方法【不想看部署方式1，就直接部署方式2】

部署方式1：代码在 [这里](https://github.com/chinesejie/gogogo) ，这是个golang的项目，使用beego框架，安装起来也比较简单，就是通常的golang项目的编译方式

```
`mkdir -p $GOPATH/src/github.com/dinp # 假设你已经配置好了GOROOT和GOPATH
cd $GOPATH/src/github.com/dinp
git clone https://github.com/dinp/builder.git
cd builder
go get ./...
# 上面那个傻逼命令一定会有问题的。。此处强调beego用1.4.1版本，go-sql-driver/mysql用1.1版本
# 使用cd github.com/astaxie/beego && git checkout -b tag_1.4.1 v1.4.1来导出`beego的``v1.4.1的代码
`# `使用cd github.com/`go-sql-driver/mysql` && git checkout -b tag_1.1 v1.1来导出```mysql的````v1.1的代码``
go build
mv conf/app.conf.example conf/app.conf
# modify conf/app.conf
## 很多值要修改。。。

#简单解释一下conf/app.conf各项配置的作用

#appname、httpport没啥好说的
#runmode取值是dev或者prod，可以参看beego的文档
#db打头的是数据库配置，数据库初始化脚本在schema.sql
#tmpdir、logdir是就是一些临时目录，不解释
#buildtimeout是编译超时时间，超过了这个时间就会被kill，单位是分钟
#uicinternal、uicexternal是UIC的配置，为啥分成两个呢？Builder和UIC通常是在一个内网的，相互之间的访问可以走内网，所以有个uicinternal，但是有的时候UIC的内网地址用户是没法访问的，所以在sso登录的时候还是需要redirect到UIC的外网地址
#registry是docker私有源，读者可以使用docker registry搭建
#buildscript是Builder内部用到的一个脚本文件地址，默认配置即可
#tplmapping这个很重要，这是base image的配置，在registry中增加了base image，也要在此配置一下，这样用户才能在页面上看到
#token，这是与UIC通信的凭证，与UIC的token配置成一样即可
#最后是启动命令；
./builder`
```

> @@@@@@@@@>>>>git 怎么checkout tag>>>>@@@@@@@@@

先 git clone 整个仓库，然后 git checkout tag_name 就可以取得 tag 对应的代码了。
但是这时候 git 可能会提示你当前处于一个“detached HEAD" 状态，因为 tag 相当于是一个快照，是不能更改它的代码的，如果要在 tag 代码的基础上做修改，你需要一个分支：

```
git checkout -b branch_name tag_name
```


快捷路径==部署方式2：如果上面的步骤太烦，直接看 下面的快捷方式

```
cd /root/gogogo #前面的代码已经下载过了
cd src/github.com/dinp/builder
go build
nohup ./builder &  ##即可启动builder，端口是8788。。

```

## 8. 安装java版本的 dashboard[同uic，需要数据库dash]

这个dashboard 可以部署images。最简单的部署方式：

```
`# 换成我的github是 https://github.com/chinesejie/gogogo下面的dash跟dash-tomcat
# 注意根路径是/home/pay/dash
cd /home/pay
git clone `https://github.com/chinesejie/`gogogo`
cd `gogogo/java`
mv dash ../..
mv dash-tomcat ../..
vi ../../dash`````/web/WEB-INF/config.txt ```````# 记得修改下`dash/web/WEB-INF/config.txt```
cd ../../dash-tomcat/bin && sh start.sh  # 默认端口8180``
```

下面说下dashboard的配置：：

```
jdbcUrl = jdbc:mysql://xxx:8306/dash   #数据库dash，文件在dash/scripts/schema.sql里面
user = root
password = 123456
devMode = false
memcacheAddrs = 127.0.0.1:11211
memcachePrefix = dash:
uicInternal = http://110.95.241.31:8080     #uic的内网
uicExternal = http://110.95.241.31:8080     #uic的外网
builder = http://110.95.241.31:8788         #builder的连接
server = http://110.95.241.31:8980          #server的连接
paasDomain = apps.io
token = token1 # UIC_TOKEN、DOMAIN这两个配置需要和之前启动其他容器时设置的参数值一致。
```

直接访问外网IP:8180，就会自动以root身份登录dashboard系统【经过uic系统认证的单点认证的】。

## 9. server【不想看部署方式1，就直接部署方式2】

部署方式1：

```
`cd $GOPATH/src/github.com/dinp
git clone https://github.com/dinp/server.git
cd `server`
go get ./...
# 上面那个傻逼命令一定会有问题的。。此处强调go-dockerclient用  git reset --hard 93458fdcb8791a25f61681b05b8aaba8018293d4
# 使用cd `github.com/`fsouza/`go-dockerclient` && git reset --hard 93458fdcb8791a25f61681b05b8aaba8018293d4来导出 特定`代码
`go build
`mv cfg.example.json cfg.json
###TODO 修改配置，，使用dash的数据库
./server
```

部署方式2：[已经git clone https://github.com/chinesejie/gogogo了]

```
`cd /root/gogogo/src/github.com/dinp
cd `server`
go get ./...
go build`
./server
```

配置文件说明cfg.json：

```
{
    "debug": true,
    "interval": 5,
    "dockerPort": 2375,  
    "domain": "apps.io",
    "localIp": "",
    "redis": {
        "dsn":"127.0.0.1:6379",  #需要拿来放信息
        "maxIdle": 10,
        "rsPrefix": "/rs/",
        "cnamePrefix": "/cname/"
    },
    "db": { #使用dashshu'ju'ku
        "dsn": "root:123456@tcp(10.101.22.19:8306)/dash?loc=Local&parseTime=true",
        "maxIdle": 2
    },
    "scribe": {#暂时可忽略。
        "ip": "10.5.6.7",
        "port": 1463
    },
    "http": {
        "addr": "0.0.0.0",
        "port": 8980
    },
    "rpc": {
        "addr": "0.0.0.0",
        "port": 8970
    }
}
```

## 10. agent【不想看部署方式1，就直接部署方式2】

部署方式1：

```
`cd $GOPATH/src/github.com/dinp
git clone https://github.com/dinp/agent
cd `agent`
go get ./...
# 上面那个傻逼命令一定会有问题的。。此处强调go-dockerclient用  git reset --hard 93458fdcb8791a25f61681b05b8aaba8018293d4
# 使用cd `github.com/`fsouza/`go-dockerclient` && git reset --hard 93458fdcb8791a25f61681b05b8aaba8018293d4来导出 特定`代码
`go build
`mv cfg.example.json cfg.json
###TODO 修改配置，，使用dash的数据库
./agent
```

部署方式2：[已经git clone https://github.com/chinesejie/gogogo了]

```
`cd /root/gogogo/src/github.com/dinp
cd agent
`go get ./...`
go build`
./agent
```

配置文件说明cfg.json  for agent：

```
{
    "debug": true,
    "localIp": "10.94.240.35",  #本地ip
    "servers": [
        "127.0.0.1:8970"      #server的连接
    ],
    "interval": 5,
    "timeout": 1000,
    "docker": "unix:///var/run/docker.sock" #可以是tcp或者native socket
}
```

## 11.goroute【不想看部署方式1，就直接部署方式2】

gorouter 是最麻烦的。因为
1）`cloudfoundry/dropsonde 代码有变化`
2）``code.google.com/``gogoprotobuf不见了。``
部署方式1：

```
`cd $GOPATH/src/github.com/dinp
git clone https://github.com/dinp/gorouter
cd gorouter
go get ./...
# 上面那个傻逼命令一定会有问题的。。此处强调 github.com/cloudfoundry/dropsonde用  git reset --hard 7d86dc6d1293ead3a00866030f77ba83577682e9
# 使用cd github.com/cloudfoundry/dropsonde && git reset --hard 7d86dc6d1293ead3a00866030f77ba83577682e9来导出 特定`代码
`// 然后用“
//sed -i 's/code\.google\.com\/p\/gogoprotobuf/github\.com\/gogo\/protobuf/g' `grep code\.google\.com\/p\/gogoprotobuf -rl ./`
//” 这句话来替换`code.google.com/``gogoprotobuf的代码。。。`

go build
`mv example_config/example.yml config/router.yml

###TODO 修改配置，，使用redis
./gorouter -c=config/router.yml
```


部署方式2：[已经git clone https://github.com/chinesejie/gogogo了]

```
`cd /root/gogogo/src/github.com/dinp
cd gorouter
`go get ./...`
go build`
./gorouter -c=config/router.yml
```

配置文件说明rouer.yml：

```
#config/router.yml
status:
  port: 8082     #状态查询端口
  user: router
  pass: routerPass

logging:
  file: router.log
  syslog:
  level: debug

port: 80  #很关键，路由入口的端口s
index: 0

pidfile: router.pid
go_max_procs: 8

redis_server: "127.0.0.1:6379"  #redis 
reload_uri_interval: 5
```


gorouter 就是cloudfoundry开源的那个V2版本的货。。

小辉同学用了redis做了改造。所以请使用全新的redis-server，并且保证这个redis的keys *有一个值，不然会启动报错。

```
$redis-cli
>rpush /rs/demo.xae.xiaomi.com 10.201.37.5:10005
>OK
```

这样就不会启动报错了。。。

这里做了两个端口映射，容器中的8082端口是状态查询端口，可以用来查看路由表、健康检查状态等，80端口则是用户请求入口。  


## 12. 补充HM。。。

现在所有的必备的容器都已经启动完毕（还有一个HM健康检查模块没起，但不影响使用），下面开始使用Dinp创建第一个container。

## 13. 构建php基础镜像。。。

部署方式1：

```
cd /root/dinp \
&& git clone https://github.com/dinp/Dockerfile.git
```

或者部署方式2：[已经git clone https://github.com/chinesejie/gogogo到/root了]

```
cd /root/dinp
mv /root/gogogo/tools/Dockerfile .
```


这个Dockerfile项目是Dinp各个基础镜像的Dockerfile，我们先来构建一个php的基础镜像。

cd Dockerfile/php，编辑build文件，将REGISTRY=registry.com:5000改成REGISTRY=内网IP:5000，然后就可以通过./build来构建镜像。这个build文件会在镜像构建完毕后将镜像push到私有registry，这就是本文最开始编辑/etc/sysconfig/docker时要加上–insecure-registry 服务器内网IP:5000的原因，否则push会失败。

./build之后，生成的docker images 可以通过dockage images查看
【可以发现 比之前的多了三个镜像nginphp、127.0.0.1:5000/nginxphp、centos】，
一开始docker-server 只有**registry**一个镜像。。

## 14.编译app应用的部署镜像


打开builder的 web界面，http://ipxxx:8788/

访问外网IP:8788进入Builder平台，app名称填app5，app版本填1，备忘留空，from base image选择php，代码可以通过提供http下载链接或者直接上传。注意一下，php的代码目录必须包含htdocs，入口文件应该放在这个目录里。我这里提供一个测试的包，里面就一个文件，显示phpinfo，[代码包链接](https://raw.githubusercontent.com/chinesejie/gogogo/master/tools/index.tar.gz)[tools下面的index.tar.gz]。之后点击build按钮即可。build页面不会自动刷新，需要自己手动刷，等页面顶部的蓝字显示successful就代表构建成功。点击右上角的History按钮，可以看到这个镜像的地址，记一下，之后要用到。[images地址举例：127.0.0.1:5000/root/app5:1]

 127.0.0.1:5000/root/app5:1表示可以从127.0.0.1:5000下到镜像，其中root/app5是repository仓库名字，而1是TAG。

TAG在docker是什么，这里就不多说了。。docker的只是自己去补充。。

再观察下 docker images ，发现又多了两个镜像【root/app5 跟 127.0.0.1/root/app5，仔细观察，可以发现这两货的image id是一样的。。】
[![env ports](https://raw.githubusercontent.com/chinesejie/gogogo/master/images/docker-images.png)](https://github.com/chinesejie/gogogo)


## 15.发布发布—发布前创建一个app

访问外网IP:8180，进入dashboard平台。点击“create app”按钮，Dinp设定每一个app都应该属于UIC中的一个Team，所以需要先在UIC里创建一个Team，**_在Dashboard页面上点击“Create Team”会自动前往UIC[外网IP:8080]创建Team页面，_**Team名称填team1，成员加上root，点击“创建”即可。回到创建App页面，刷新之后就可以看到有team1这个Team了，App名字填app5，Health interface留空即可。
[![env ports](https://raw.githubusercontent.com/chinesejie/gogogo/master/images/dash-application.png)](https://github.com/chinesejie/gogogo)
## 16. 发布发布—发布时候的deploy

点击上图的dashboard的deploy链接，
进入页面填写参数如下：内存填256MB，个数填写1，images填写127.0.0.1:5000/root/app5💯 
点击submit，就能发布了。。如下图：
[![env ports](https://raw.githubusercontent.com/chinesejie/gogogo/master/images/dash-deploy.png)](https://github.com/chinesejie/gogogo)

1）搭建服务器bind 120.95.241.31 ，然后把*.apps.io的泛域名A记录到实验机器上110.95.241.31 。

2）在实验机器上110.95.241.31 上vi/etc/resolv.conf，设置nameserver 为bind地址bind 120.95.241.31 。

3）实验机器上110.95.241.31 上 访问[http://app5.apps.io](http://app5.apps.io/) ，这时候由于实验机器的dns指向bind机器，这时候解析*.apps.io为110.95.241.31，所以就 会路由到我们亲爱的gorouter上。。

4）
这个时候gorouter 根据redis里面的 路由表，开始 把请求转发到到对应的agent机器上的 docker容器上去了。也就是说 访问app5.apps.io，应该就能看到phpinfo页面了。

送出命令：
curl --url http://app5.apps.io


## 16. 扩容

在Dashboard页面，点击app1右边的scale，instance count填2，点击scale按钮。点refresh按钮，很快就能看到两个实例出来了。
访问app5.apps.io，不断刷新，可以看到System那一栏显示的机器名会改变[其实在同一个机器也看不到变化，可以到两个docker容器 里面去 nginx的access日志]，说明我们的请求会被随机分发到两个container中的一个。


## 17、docker命令

PS：docker 命令【1.7.1版本】

```
docker ps -l #-l，查看最新创建的容器 
docker ps -a #-a，查看所有容器包括停止状态的容器 
docker kill fd3c0c622af6 #停止正在运行的容器
docker rm fd3c0c622af6 #停止容器
docker ps #查看所有正在运行中的容器，不加参数
docker pull `registry`#从远端拉取一个image
# github.com/cloudfoundry/dropsonde用这个版本2cc29e4670b8205a1c25e751cf9ffb38813eba28

```

## 18、怎么搭建bind机器。。start github，然后mail找我，chinesejie#[qq.com](http://qq.com/)

## 19、其他。。

1）最好是在goRouter前面架设LVS，dns配置到LVS。
2）日志手机用filebeat..

20.。。谢谢大家支持。。
