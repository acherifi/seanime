package codegen

import (
	"cmp"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	typescriptFileName = "types.ts"
)

// GenerateTypescriptFile generates a Typescript file containing the types for the API routes parameters and responses based on the Docs struct.
func GenerateTypescriptFile(docsFilePath string, publicStructsFilePath string, outDir string, goStructStrs []string) {

	handlers := LoadHandlers(docsFilePath)

	goStructs := LoadPublicStructs(publicStructsFilePath)

	// e.g. map["models.User"]*GoStruct
	goStructsMap := make(map[string]*GoStruct)
	for _, goStruct := range goStructs {
		goStructsMap[goStruct.Package+"."+goStruct.Name] = goStruct
	}

	// Expand the structs with embedded structs
	for _, goStruct := range goStructs {
		for _, embeddedStructType := range goStruct.EmbeddedStructTypes {
			if embeddedStructType != "" {
				if usedStruct, ok := goStructsMap[embeddedStructType]; ok {
					for _, usedField := range usedStruct.Fields {
						goStruct.Fields = append(goStruct.Fields, usedField)
					}
				}
			}
		}
	}

	// Create the typescript file
	_ = os.MkdirAll(outDir, os.ModePerm)
	file, err := os.Create(filepath.Join(outDir, typescriptFileName))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write the typescript file
	file.WriteString("// This file was generated by the Seanime API documentation generator.\n\n")

	// Get all the returned structs from the routes
	// e.g. @returns models.User
	structStrMap := make(map[string]int)
	for _, str := range goStructStrs {
		if _, ok := structStrMap[str]; ok {
			structStrMap[str]++
		} else {
			structStrMap[str] = 1
		}
	}
	for _, handler := range handlers {
		if handler.Api != nil {
			switch handler.Api.ReturnTypescriptType {
			case "null", "string", "number", "boolean":
				continue
			}

			if _, ok := structStrMap[handler.Api.ReturnGoType]; ok {
				structStrMap[handler.Api.ReturnGoType]++
			} else {
				structStrMap[handler.Api.ReturnGoType] = 1
			}
		}
	}

	// Isolate the structs that are returned more than once
	sharedStructStrs := make([]string, 0)
	otherStructStrs := make([]string, 0)

	for k, v := range structStrMap {
		if v > 1 {
			sharedStructStrs = append(sharedStructStrs, k)
		} else {
			otherStructStrs = append(otherStructStrs, k)
		}
	}

	// Now that we have the returned structs, store them in slices
	sharedStructs := make([]*GoStruct, 0)
	otherStructs := make([]*GoStruct, 0)

	for _, structStr := range sharedStructStrs {
		// e.g. "models.User"
		structStrParts := strings.Split(structStr, ".")
		if len(structStrParts) != 2 {
			continue
		}

		// Find the struct
		goStruct, ok := goStructsMap[structStr]
		if ok {
			sharedStructs = append(sharedStructs, goStruct)
		}

	}
	for _, structStr := range otherStructStrs {
		// e.g. "models.User"
		structStrParts := strings.Split(structStr, ".")
		if len(structStrParts) != 2 {
			continue
		}

		// Find the struct
		goStruct, ok := goStructsMap[structStr]
		if ok {
			otherStructs = append(otherStructs, goStruct)
		}
	}

	//-------------------------

	referencedStructs, ok := getReferencedStructsRecursively(sharedStructs, otherStructs, goStructsMap)
	if !ok {
		panic("Failed to get referenced structs")
	}

	// Keep track of written Typescript types
	// This is to avoid name collisions
	writtenTypes := make(map[string]*GoStruct)

	// Group the structs by package
	structsByPackage := make(map[string][]*GoStruct)
	for _, goStruct := range referencedStructs {
		if _, ok := structsByPackage[goStruct.Package]; !ok {
			structsByPackage[goStruct.Package] = make([]*GoStruct, 0)
		}
		structsByPackage[goStruct.Package] = append(structsByPackage[goStruct.Package], goStruct)
	}

	packages := make([]string, 0)
	for k := range structsByPackage {
		packages = append(packages, k)
	}

	slices.SortStableFunc(packages, func(i, j string) int {
		return cmp.Compare(i, j)
	})

	file.WriteString("export type Nullish<T> = T | null | undefined\n\n")

	for _, pkg := range packages {

		file.WriteString("//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////\n")
		file.WriteString(fmt.Sprintf("// %s\n", strings.ReplaceAll(cases.Title(language.English, cases.Compact).String(strings.ReplaceAll(pkg, "_", " ")), " ", "")))
		file.WriteString("//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////\n\n")

		structs := structsByPackage[pkg]
		slices.SortStableFunc(structs, func(i, j *GoStruct) int {
			return cmp.Compare(i.FormattedName, j.FormattedName)
		})

		// Write the shared structs first
		for _, goStruct := range structs {

			writeTypescriptType(file, goStruct, writtenTypes)

		}

	}

	//for _, goStruct := range referencedStructs {
	//
	//	writeTypescriptType(file, goStruct, writtenTypes)
	//
	//}

}

// getReferencedStructsRecursively returns a map of GoStructs that are referenced by the fields of sharedStructs and otherStructs.
func getReferencedStructsRecursively(sharedStructs, otherStructs []*GoStruct, goStructsMap map[string]*GoStruct) (map[string]*GoStruct, bool) {
	allStructs := make(map[string]*GoStruct)
	for _, sharedStruct := range sharedStructs {
		allStructs[sharedStruct.Package+"."+sharedStruct.Name] = sharedStruct
	}
	for _, otherStruct := range otherStructs {
		allStructs[otherStruct.Package+"."+otherStruct.Name] = otherStruct
	}

	// Keep track of the structs that have been visited

	referencedStructs := make(map[string]*GoStruct)

	for _, strct := range allStructs {
		getReferencedStructs(strct, referencedStructs, goStructsMap)
	}

	return referencedStructs, true
}

func getReferencedStructs(goStruct *GoStruct, referencedStructs map[string]*GoStruct, goStructsMap map[string]*GoStruct) {
	if _, ok := referencedStructs[goStruct.Package+"."+goStruct.Name]; ok {
		return
	}
	referencedStructs[goStruct.Package+"."+goStruct.Name] = goStruct
	for _, field := range goStruct.Fields {
		if field.UsedStructType != "" {
			if usedStruct, ok := goStructsMap[field.UsedStructType]; ok {
				getReferencedStructs(usedStruct, referencedStructs, goStructsMap)
			}
		}
	}
	if goStruct.AliasOf != nil {
		if usedStruct, ok := goStructsMap[goStruct.AliasOf.UsedStructType]; ok {
			getReferencedStructs(usedStruct, referencedStructs, goStructsMap)
		}
	}
}

func writeTypescriptType(f *os.File, goStruct *GoStruct, writtenTypes map[string]*GoStruct) {
	f.WriteString("/**\n")
	f.WriteString(fmt.Sprintf(" * - Filepath: %s\n", strings.TrimPrefix(goStruct.Filepath, "../")))
	f.WriteString(fmt.Sprintf(" * - Filename: %s\n", goStruct.Filename))
	f.WriteString(fmt.Sprintf(" * - Package: %s\n", goStruct.Package))
	if len(goStruct.Comments) > 0 {
		f.WriteString(fmt.Sprintf(" * @description\n"))
		for _, cmt := range goStruct.Comments {
			f.WriteString(fmt.Sprintf(" *  %s\n", strings.TrimSpace(cmt)))
		}
	}
	f.WriteString(" */\n")

	if len(goStruct.Fields) > 0 {
		f.WriteString(fmt.Sprintf("export type %s = {\n", goStruct.FormattedName))
		for _, field := range goStruct.Fields {
			fieldNameSuffix := ""
			if !field.Required {
				fieldNameSuffix = "?"
			}

			if len(field.Comments) > 0 {
				f.WriteString(fmt.Sprintf("    /**\n"))
				for _, cmt := range field.Comments {
					f.WriteString(fmt.Sprintf("     * %s\n", strings.TrimSpace(cmt)))
				}
				f.WriteString(fmt.Sprintf("     */\n"))
			}

			typeText := field.TypescriptType
			//if !field.Required {
			//	switch typeText {
			//	case "string", "number", "boolean":
			//	default:
			//		typeText = "Nullish<" + typeText + ">"
			//	}
			//}

			f.WriteString(fmt.Sprintf("    %s%s: %s\n", field.JsonName, fieldNameSuffix, typeText))
		}
		f.WriteString("}\n\n")
	}

	if goStruct.AliasOf != nil {
		if goStruct.AliasOf.DeclaredValues != nil && len(goStruct.AliasOf.DeclaredValues) > 0 {
			union := ""
			if len(goStruct.AliasOf.DeclaredValues) > 5 {
				union = strings.Join(goStruct.AliasOf.DeclaredValues, " |\n    ")
			} else {
				union = strings.Join(goStruct.AliasOf.DeclaredValues, " | ")
			}
			f.WriteString(fmt.Sprintf("export type %s = %s\n\n", goStruct.FormattedName, union))
		} else {
			f.WriteString(fmt.Sprintf("export type %s = %s\n\n", goStruct.FormattedName, goStruct.AliasOf.TypescriptType))
		}
	}

	// Add the struct to the written types
	writtenTypes[goStruct.Package+"."+goStruct.Name] = goStruct
}

func getUnformattedGoType(goType string) string {
	if strings.HasPrefix(goType, "[]") {
		return getUnformattedGoType(goType[2:])
	}

	if strings.HasPrefix(goType, "*") {
		return getUnformattedGoType(goType[1:])
	}

	if strings.HasPrefix(goType, "map[") {
		s := strings.TrimPrefix(goType, "map[")
		value := ""
		for i, c := range s {
			if c == ']' {
				value = s[i+1:]
				break
			}
		}
		return getUnformattedGoType(value)
	}

	return goType
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
