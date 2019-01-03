# codeGenerator
数据库表转为golang实体struct的工具

# USEAGE:
    Builder().Dns("root:root@tcp(localhost:3306)/schema?charset=utf8").Schema("schema").Tags("gorm:\"自定义的标签,1,2\",validate:\"NOTNULL,Min:1\"").PackageName("generator").ModelPath(path.Join("C:", "generator")).Build().Generate()
