package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type tokenType int

const (
	show        tokenType = 0
	hide        tokenType = 1
	wait        tokenType = 2
	defaultWait           = 1000
)

type token struct {
	Type tokenType
	Id   string
	Wait int
}

func parseShow(line string) token {
	tokens := strings.Split(line, " ")
	return token{
		Type: show,
		Id:   tokens[1],
		Wait: defaultWait,
	}
}

func parseHide(line string) token {
	tokens := strings.Split(line, " ")
	return token{
		Type: hide,
		Id:   tokens[1],
		Wait: defaultWait,
	}
}

func parseWait(line string) token {
	tokens := strings.Split(line, " ")

	val, err := strconv.Atoi(tokens[1])
	if err != nil {
		fmt.Printf("Error parsing wait: %s", err.Error())
	}
	return token{
		Type: wait,
		Wait: val,
	}
}

func main() {
	// inputSvgFile := "UpdateEvent.svg"
	// inputAnimation := "input"
	// outputSvgFile := "result.svg"

	if len(os.Args) < 3 {
		fmt.Printf("args needed: Input SVG, Input Animation, Output SVG\n")
		return
	}

	inputSvgFile := os.Args[1]
	inputAnimation := os.Args[2]
	outputSvgFile := os.Args[3]

	data, _ := os.ReadFile(inputAnimation)
	reader := bufio.NewReader(bytes.NewReader(data))

	tokens := make([]token, 0)

	for {
		lineBytes, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		line := string(lineBytes)
		if strings.HasPrefix(line, "show") {
			tokens = append(tokens, parseShow(line))
		}

		if strings.HasPrefix(line, "hide") {
			tokens = append(tokens, parseHide(line))
		}

		if strings.HasPrefix(line, "wait") {
			tokens = append(tokens, parseWait(line))
		}
	}

	for _, t := range tokens {
		fmt.Printf("%v\n", t)
	}

	step, totalMs := calculateStep(tokens)
	fmt.Printf("step: %v\n", step)
	fmt.Printf("totalMs: %v\n", totalMs)

	keyframes := calculateKeyframes(tokens, step)
	css := generateCssAnimation(keyframes, totalMs)

	svgData, _ := os.ReadFile(inputSvgFile)
	svg := addCssToSvg(string(svgData), css)

	os.WriteFile(outputSvgFile, []byte(svg), os.ModePerm)
}

func calculateStep(tokens []token) (float32, int) {
	totalMs := 0
	for _, t := range tokens {
		if t.Type == wait {
			totalMs += t.Wait
		}
	}

	return 1000.0 / float32(totalMs) * 100.0, totalMs
}

type keyframe struct {
	Stage   float32
	Opacity float32
}

func calculateKeyframes(tokens []token, step float32) map[string][]keyframe {
	currentStage := float32(0.0)
	nextStage := currentStage + step

	keyframes := make(map[string][]keyframe)

	for _, t := range tokens {
		if t.Type == show || t.Type == hide {
			if _, ok := keyframes[t.Id]; !ok {
				keyframes[t.Id] = make([]keyframe, 0)
				keyframes[t.Id] = append(keyframes[t.Id], keyframe{Stage: 0, Opacity: 0})
			}
		}
		switch t.Type {
		case wait:
			currentStage += float32(t.Wait) / 1000.0 * step
			nextStage = currentStage + step
		case show:
			keyframes[t.Id] = append(keyframes[t.Id], keyframe{Stage: currentStage, Opacity: 0})
			keyframes[t.Id] = append(keyframes[t.Id], keyframe{Stage: nextStage, Opacity: 1})
		case hide:
			keyframes[t.Id] = append(keyframes[t.Id], keyframe{Stage: currentStage, Opacity: 1})
			keyframes[t.Id] = append(keyframes[t.Id], keyframe{Stage: nextStage, Opacity: 0})
		}
	}

	for k, v := range keyframes {
		keyframes[k] = append(keyframes[k], keyframe{Stage: 100.0, Opacity: v[len(v)-1].Opacity})
	}

	return keyframes
}

type Animation struct {
	ObjectId string
	Name     string
	TotalMs  int
}

type KeyframesCss struct {
	AnimationName string
	Keyframes     []keyframe
}

func generateCssAnimation(keyframes map[string][]keyframe, totalMs int) string {
	var builder strings.Builder
	writer := io.Writer(&builder)

	builder.WriteString("<style>")

	animationTemplate, err := template.New("animation").Parse("#{{ .ObjectId }} { animation: {{ .Name }} {{ .TotalMs}}ms linear infinite normal forwards; } ")
	if err != nil {
		fmt.Printf("Error parsing Animation Template: %s", err.Error())
	}

	keyframesTemplate, err := template.New("keyframes").Parse("@keyframes {{ .AnimationName }} " +
		"{ {{ range .Keyframes }}{{ .Stage }}% { opacity: {{ .Opacity }}; } {{end}}}")
	if err != nil {
		fmt.Printf("Error parsing Animation Template: %s", err.Error())
	}

	objectIdPrfix := "cell-"
	animationNamePrefix := "anim"
	animationCounter := 0

	for k, v := range keyframes {
		animName := fmt.Sprintf("%s%d", animationNamePrefix, animationCounter)
		anim := Animation{
			ObjectId: objectIdPrfix + k,
			Name:     animName,
			TotalMs:  totalMs,
		}
		animationTemplate.Execute(writer, anim)
		animationCounter++

		keyframesCss := KeyframesCss{
			AnimationName: animName,
			Keyframes:     v,
		}

		keyframesTemplate.Execute(writer, keyframesCss)
	}

	builder.WriteString("</style>")
	return builder.String()
}

func addCssToSvg(svg string, css string) string {
	index := strings.Index(svg, "<svg")
	for svg[index] != '>' {
		index++
	}
	index++
	svg = svg[:index] + css + svg[index:]
	return svg
}
