// package main
//
// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// )
//
// func main() {
// 	fmt.Println("Enter the absolute path to an output directory ")
// 	var outputDir string
// 	n, err := fmt.Scanln(&outputDir)
// 	if err != nil {
// 		log.Fatalf("could not read the input : %s", err.Error())
// 		return
// 	}
// 	if n == 0 {
// 		log.Fatal("no output directory has been given")
// 		return
// 	}
// 	currentWorkingDir, err := os.Getwd()
// 	if err != nil {
// 		log.Fatalf("failed to fetch current working directory : %s", err.Error())
// 		return
// 	}
// 	outputFile := filepath.Join(outputDir, "compiled-binary")
// 	cmd := exec.Command("go", "build", "-o", outputFile)
// 	cmd.Dir = currentWorkingDir
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	err = cmd.Run()
// 	if err != nil {
// 		fmt.Printf("failed to execute command : %s", err.Error())
// 		return
// 	}
// }
