package generator

import (
	"path"
	"testing"
)

func TestGenerator_Generate(t *testing.T) {
	Builder().Dns("root:root@tcp(192.168.3.149:3306)/tanbaye_cec?charset=utf8").Schema("tanbaye_cec").Tags("gorm:\"自定义的标签, 1,2\",validate:\"NOTNULL,Min:1\"").PackageName("generator").ModelPath(path.Join("F:", "Go_demo", "codeGenerator")).Build().Generate()
}
