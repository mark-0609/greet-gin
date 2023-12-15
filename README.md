

### 技术栈
gorm gin jaeger rabbitmq kibana mysql redis elasticsearch
### 注意点

* bin目录需有执行权限，data存放的是mysql和es数据，bin目录存放的是modd编译后的可执行文件
。在根目录下执行：
```shell
    mkdir bin/data && chmod -R 777 bin
```
* copy一份config目录的app.init.exmaple 文件，改为app.ini
```shell
    cp config/app.ini.exmaple config/app.ini
```
* 本地工具连接mysql的话要先进入容器，给root设置下远程连接权限

```shell
 docker exec -it mysql mysql -uroot -p ##输入密码：PXDN93VRKUm8TeE7
 use mysql;
 update user set host='%' where user='root';
 FLUSH PRIVILEGES;
 create database greet default character set utf8mb4 collate utf8mb4_unicode_ci;
```
* modd.conf :  modd热加载配置文件，关于modd更多用法 ： https://github.com/cortesi/modd
* docker，启动 ~ docker-compose up -d


------------------

nginx: http://127.0.0.1:8888/

jaeger: http://127.0.0.1:16686/search

kibana: http://127.0.0.1:5601/

Mysql :  自行客户端工具(Navicat、Sequel Pro)查看

- host : 127.0.0.1

- port : 33069

- username : root

- pwd : PXDN93VRKUm8TeE7

Redis :  自行工具（redisManager）查看

- host : 127.0.0.1

- port : 36379

- pwd : G62m50oigInC30sf