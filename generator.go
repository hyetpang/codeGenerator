package generator

import (
	"database/sql"
	"fmt"
	_ "github.com/Go-SQL-Driver/MySQL"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type Generator struct {
	// 指定生成的表名，不指定默认生成所有
	tables []string
	// 数据库
	schema string
	// 是否添加json标签,默认true添加
	isAddJsonTag bool
	// 自定义的tag
	tags string
	// 生成的model路径
	modelPath string
	// 数据库链接dns
	dns string
	// 是否把所有的model都存放在一个文件,默认true
	isSingleFile bool
	// 如果表名是下划线组成，那么遇到下划线的地方就转为大写
	covertUnderlineToUpper bool
	// 包名
	packageName string
}

type CodeBuilder struct {
	// 指定生成的表名，不指定默认生成所有
	tables []string
	// 数据库
	schema string
	// 是否添加json标签,默认true添加
	isAddJsonTag bool
	// 自定义的tag
	tags string
	// 生成的model路径
	modelPath string
	// 数据库链接dns
	dns string
	// 是否把所有的model都存放在一个文件,默认true
	isSingleFile bool
	// 如果表名是下划线组成，那么遇到下划线的地方就转为大写
	covertUnderlineToUpper bool
	// 包名
	packageName string
}

type modelInfo struct {
	TableName     string
	ColumnName    string
	DataType      string
	ColumnComment string
}

func (codeBuilder *CodeBuilder) Tables(tables []string) *CodeBuilder {
	codeBuilder.tables = tables
	return codeBuilder
}

func (codeBuilder *CodeBuilder) Schema(schema string) *CodeBuilder {
	if len(schema) == 0 {
		panic("请输入有效的schema!")
	}
	codeBuilder.schema = schema
	return codeBuilder
}

func (codeBuilder *CodeBuilder) PackageName(packageName string) *CodeBuilder {
	if len(packageName) == 0 {
		panic("请输入有效的packageName!")
	}
	codeBuilder.packageName = packageName
	return codeBuilder
}

func (codeBuilder *CodeBuilder) IsAddJsonTag(isAddJsonTag bool) *CodeBuilder {
	codeBuilder.isAddJsonTag = isAddJsonTag
	return codeBuilder
}

func (codeBuilder *CodeBuilder) Tags(tags string) *CodeBuilder {
	codeBuilder.tags = tags
	return codeBuilder
}

func (codeBuilder *CodeBuilder) ModelPath(modelPath string) *CodeBuilder {
	// 判断路径是否存在
	fileInfo, err := os.Stat(modelPath)
	if err == nil && fileInfo.IsDir() {
		codeBuilder.modelPath = modelPath
		return codeBuilder
	}
	if os.IsNotExist(err) || !fileInfo.IsDir() {
		panic("model路径有误,请指定一个有效的路径!")
	}
	panic(fmt.Sprintf("model路径有误:%s", err))
}

func (codeBuilder *CodeBuilder) Dns(dns string) *CodeBuilder {
	codeBuilder.dns = dns
	return codeBuilder
}

func (codeBuilder *CodeBuilder) CovertUnderlineToUpper(covertUnderlineToUpper bool) *CodeBuilder {
	codeBuilder.covertUnderlineToUpper = covertUnderlineToUpper
	return codeBuilder
}

func Builder() *CodeBuilder {
	generator := &CodeBuilder{
		isAddJsonTag:           true,
		isSingleFile:           true,
		covertUnderlineToUpper: true,
	}
	return generator
}

func (codeBuilder *CodeBuilder) Build() *Generator {
	// 验证
	if len(codeBuilder.modelPath) <= 0 {
		panic("请输入生成的model的存放路径，路径是一个目录")
	}
	if len(codeBuilder.dns) <= 0 {
		panic("请输入数据库链接的dns")
	}
	if len(codeBuilder.schema) <= 0 {
		panic("请输入数据库schema")
	}
	generator := &Generator{
		tables:                 codeBuilder.tables,
		isAddJsonTag:           codeBuilder.isAddJsonTag,
		schema:                 codeBuilder.schema,
		tags:                   codeBuilder.tags,
		modelPath:              codeBuilder.modelPath,
		dns:                    codeBuilder.dns,
		isSingleFile:           codeBuilder.isSingleFile,
		covertUnderlineToUpper: codeBuilder.covertUnderlineToUpper,
		packageName:            codeBuilder.packageName,
	}
	return generator
}

func (generator *Generator) Generate() {
	// 链接数据库
	mysqlDb, err := sql.Open("mysql", generator.dns)
	if err != nil {
		panic(fmt.Sprintf("数据库连接失败:%s\r\n给定的dns:%s", err, generator.dns))
	}
	var tables = ""
	if len(generator.tables) > 0 {
		for _, v := range generator.tables {
			tables += fmt.Sprintf("and TABLE_NAME in (\"%s\")", v)
		}
	}
	schemaQuerySQL := fmt.Sprintf("select TABLE_NAME, COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT from information_schema.COLUMNS where TABLE_SCHEMA=\"%s\" %s order by TABLE_NAME", generator.schema, tables)
	rows, err := mysqlDb.Query(schemaQuerySQL)
	if err != nil {
		panic(fmt.Sprintf("获取数据库信息失败:%s", err))
	}
	var tableName, columnName, dataType, columnComment string
	model := make(map[string][]modelInfo)
	// 组装给定schema的所有数据
	for rows.Next() {
		err := rows.Scan(&tableName, &columnName, &dataType, &columnComment)
		if err != nil {
			panic(fmt.Sprintf("获取表信息失败:%s", err))
		}
		var getDataType = GetDataType[dataType]
		if len(getDataType) <= 0 {
			fmt.Println(columnName, getDataType)
			getDataType = dataType
		}
		modelData := modelInfo{
			TableName:     tableName,
			ColumnComment: columnComment,
			ColumnName:    columnName,
			DataType:      getDataType,
		}
		models, ok := model[tableName]
		if ok {
			models = append(models, modelData)
		} else {
			var modelTemp []modelInfo
			models = append(modelTemp, modelData)
		}
		model[tableName] = models
	}
	// 生成model
	generator.toModel(model)
}

func (generator *Generator) toModel(models map[string][]modelInfo) {
	var firstPackage = "package" + TAB + generator.packageName + strings.Repeat(ENTER, 2)
	var modelStructs = ""
	for tableName, modelArr := range models {
		var modeName = tableName
		if generator.covertUnderlineToUpper {
			modeName = getName(modeName)
		}
		var modelStruct = fmt.Sprintf("type %s struct {%s", modeName, ENTER)
		//var isContainTime = false
		for _, model := range modelArr {
			var tag = ``
			var columnName = model.ColumnName
			if generator.covertUnderlineToUpper {
				columnName = getName(columnName)
			}
			if generator.isAddJsonTag {
				tag += fmt.Sprintf(`json:"%s"`, strings.ToLower(string(columnName[0]))+columnName[1:])
			}
			if len(generator.tags) > 0 {
				//temp := template.New("tags")
				//parse, err := temp.Parse(generator.tags)
				//if err != nil {
				//	fmt.Println("")
				//}
				tag += "," + strings.Replace(generator.tags, COLUMN, model.ColumnName, -1)
			}
			modelStruct += TAB + columnName + TAB + model.DataType + TAB + "`" + tag + "`" + ENTER
		}
		modelStruct += "}" + strings.Repeat(ENTER, 2)
		modelStruct += strings.Replace(strings.Replace(TABLE_NAME_FUNC, MODEL_NAME, modeName, -1), TABLE_NAME, tableName, -1)
		modelStruct += "}" + strings.Repeat(ENTER, 2)
		modelStructs += modelStruct
		modelStruct = firstPackage + modelStruct
		if !generator.isSingleFile {
			name := path.Join(generator.modelPath, modeName+".go")
			file, err := os.Create(name)
			if err != nil {
				panic("创建文件:" + name + "失败")
			}
			n, err := file.WriteString(modelStruct)
			if err != nil {
				panic("写入文件:" + name + "失败")
			}
			fmt.Println("成功!大小:" + strconv.Itoa(n))
			gofmtFile(name)
		}
	}
	if generator.isSingleFile {
		modelStructs = firstPackage + modelStructs
		name := path.Join(generator.modelPath, "model.go")
		file, err := os.Create(name)
		if err != nil {
			panic("创建文件:" + name + "失败")
		}
		n, err := file.WriteString(modelStructs)
		if err != nil {
			panic("写入文件:" + name + "失败")
		}
		fmt.Println("成功!大小:" + strconv.Itoa(n))
		gofmtFile(name)
	}
}

func getName(tableName string) string {
	modelNames := strings.Split(tableName, Underline)
	var modelName string
	for _, name := range modelNames {
		modelName += strings.Title(name)
	}
	return modelName
}

func gofmtFile(filePath string) {
	err := exec.Command("gofmt", "-l", "-w", filePath).Start()
	if err != nil {
		fmt.Printf("格式化文件[%s]失败:%s"+ENTER, filePath, err)
	}
}
