//go:build js
// +build js

package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"math"
	"net"
	"net/http"
	"syscall/js"

	"github.com/disintegration/imaging"
)

// GOOS=js GOARCH=wasm go build -o ./assets/json.wasm
func main() {
	// ll := &ListNode{Val: 1, Next: &ListNode{Val: 2, Next: &ListNode{Val: 3, Next: &ListNode{Val: 4, Next: &ListNode{Val: 5}}}}}
	fmt.Println("Go Web Assembly")
	js.Global().Set("phashGo", phashGo())
	js.Global().Set("curlGo", curlGo())
	js.Global().Set("webGo", webGo())
	js.Global().Set("tcpGo", tcpGo())

	<-make(chan bool)
}

func curlGo() js.Func {
	curlFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return curl()
	})
	return curlFunc
}

func webGo() js.Func {
	curlFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return web()
	})
	return curlFunc
}

func tcpGo() js.Func {
	curlFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return tcp()
	})
	return curlFunc
}

func tcp() string {
	go func() {
		ln, err := net.Listen("tcp", ":8089")
		if err != nil {
			fmt.Println(err)
			return
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println(err)
				return
			}

			go func() {
				for {
					reader := bufio.NewReader(conn)
					var buf [128]byte
					n, err := reader.Read(buf[:]) // 读取数据
					if err != nil {
						fmt.Println("read from client failed, err:", err)
						break
					}
					recvStr := string(buf[:n])
					fmt.Println("收到client端发来的数据：", recvStr)
					conn.Write([]byte(recvStr)) // 发送数据
				}
			}()
		}
	}()

	return "ok"
}

func web() string {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "hello\n") })
	go http.ListenAndServe(":7890", nil)
	return "ok"
}

func curl() string {
	go func() {
		resp, err := http.Get("wasm_exec.js")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		fmt.Println("http code:", resp.Status)

		res, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(res))
	}()
	return "res"
}

func phashGo() js.Func {
	jsonFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			return "Invalid no of arguments passed"
		}

		inputJSON := args[0].String()
		fmt.Printf("input %#v\n", inputJSON)

		bff, err := base64.StdEncoding.DecodeString(inputJSON)
		if err != nil {
			fmt.Printf("err %#v\n", err)
		}

		bf := bytes.NewBuffer(bff)

		return phash3(bf)
	})
	return jsonFunc
}

func phash3(f io.Reader) string {
	// 1. 读取图片文件
	img, err := imaging.Decode(f)
	if err != nil {
		panic(err)
	}

	// 2. 计算感知哈希
	phashValue, err := ComputePHash(img)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Perceptual Hash: %016x\n", phashValue)

	return fmt.Sprintf("%016x", phashValue)
}

// ComputePHash 计算图像的感知哈希值
func ComputePHash(img image.Image) (uint64, error) {
	// 步骤1: 调整大小为32x32并转为灰度图
	resized := imaging.Resize(img, 32, 32, imaging.Lanczos)
	gray := imaging.Grayscale(resized)

	// 步骤2: 转换为二维灰度矩阵
	grayMatrix := imageToGrayMatrix(gray)

	// 步骤3: 计算DCT变换
	dctMatrix := applyDCT(grayMatrix)

	// 步骤4: 获取8x8低频部分 (排除第一个系数)
	lowFreq := make([]float64, 64)
	index := 0
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if i == 0 && j == 0 {
				continue // 跳过DC系数
			}
			lowFreq[index] = dctMatrix[i][j]
			index++
		}
	}
	lowFreq = lowFreq[:63] // 取63个AC系数

	// 步骤5: 计算平均值
	mean := calculateMean(lowFreq)

	// 步骤6: 生成二进制哈希
	var hash uint64
	for i, val := range lowFreq {
		if val > mean {
			hash |= 1 << uint(63-i) // 64位哈希
		}
	}

	return hash, nil
}

// 辅助函数 ---------------------------------------------------

// 将图像转换为灰度矩阵
func imageToGrayMatrix(img image.Image) [][]float64 {
	bounds := img.Bounds()
	matrix := make([][]float64, bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := make([]float64, bounds.Dx())
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// 转换为YUV亮度值
			row[x] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
		matrix[y] = row
	}
	return matrix
}

// 离散余弦变换 (DCT-II)
func applyDCT(matrix [][]float64) [][]float64 {
	size := len(matrix)
	dct := make([][]float64, size)

	// 一维DCT变换
	dct1D := func(input []float64) []float64 {
		output := make([]float64, len(input))
		n := float64(len(input))
		for k := range output {
			var sum float64
			for i, val := range input {
				sum += val * math.Cos(math.Pi*(float64(i)+0.5)*float64(k)/n)
			}
			if k == 0 {
				output[k] = sum * math.Sqrt(1.0/n)
			} else {
				output[k] = sum * math.Sqrt(2.0/n)
			}
		}
		return output
	}

	// 先对行变换
	for y := range matrix {
		dct[y] = dct1D(matrix[y])
	}

	// 对列变换
	for x := 0; x < size; x++ {
		column := make([]float64, size)
		for y := 0; y < size; y++ {
			column[y] = dct[y][x]
		}
		transformed := dct1D(column)
		for y := 0; y < size; y++ {
			dct[y][x] = transformed[y]
		}
	}

	return dct
}

// 计算平均值（排除DC系数）
func calculateMean(data []float64) float64 {
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}
