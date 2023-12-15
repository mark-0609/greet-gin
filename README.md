
### 注意点


本地工具连接mysql的话要先进入容器，给root设置下远程连接权限

```shell
 docker exec -it mysql mysql -uroot -p ##输入密码：PXDN93VRKUm8TeE7
 use mysql;
 update user set host='%' where user='root';
 FLUSH PRIVILEGES;
```

* data和bin目录需有执行权限，data存放的是mysql和es执行数据，bin目录存放的是modd编译后的可执行文件
* copy一份config目录的app.init.exmaple 文件，改为app.ini
* modd.conf :  modd热加载配置文件，关于modd更多用法 ： https://github.com/cortesi/modd
* docker，启动 ~ docker-compose up -d