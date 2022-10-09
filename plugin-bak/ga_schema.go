package plugin_bak

import (
	"fmt"
	"io/ioutil"

	"tencent2/tools/dev_tools/t2cli/schemas/input"
)

// On 监听的事件
type On struct {
	Listeners []Listener
}

// Listener 监听的具体参数
type Listener struct {
	Event      string
	Conditions map[string]interface{}
}

// Workflow 工作流协议
type Workflow struct {
	Interaction Interaction    `yaml:"schemas"`
	On          On             `yaml:"on"`
	Scenes      []Scene        `yaml:"scenes"`
	Jobs        map[string]Job `yaml:"jobs"`
}

// Scene 描述协议场景
type Scene struct {
	// Name 场景命名
	Name string `yaml:"name"`
	// SubCommands 子命令列表，例如 tc a b，则这里设置为 ["a", "b"]
	SubCommands []string `yaml:"sub-commands"`
	// Inputs 输入组件
	Inputs input.Components `yaml:"inputs"`
}

// Job 任务协议
type Job struct {
	Name string `yaml:"name"`
}

// Interaction 交互模型
type Interaction struct{}

// ResolveWorkflow 从文件解析工作流协议
func ResolveWorkflow(path string) (*Workflow, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	workflow := &Workflow{}
	err = yaml.Unmarshal(data, workflow)
	if err != nil {
		return nil, fmt.Errorf("解析协议文件 [%s] 失败：%w", path, err)
	}
	return workflow, err
}

// UnmarshalYAML 自定义On事件yml解析
func (o *On) UnmarshalYAML(unmarshal func(interface{}) error) error {
	listeners, err := parseOnEvent(unmarshal)
	if err == nil {
		o.Listeners = listeners
		return nil
	}
	listeners, err = parseOnEvents(unmarshal)
	if err == nil {
		o.Listeners = listeners
		return nil
	}
	listeners, err = parseOnEventsWithConditions(unmarshal)
	if err != nil {
		return err
	}
	o.Listeners = listeners
	return nil
}

// parseOnEvent 此时, yaml文件事件类似为, on: pre-commit
func parseOnEvent(unmarshal func(interface{}) error) ([]Listener, error) {
	event := ""
	err := unmarshal(&event)
	if err != nil {
		return nil, err
	}
	return []Listener{{Event: event}}, nil
}

// parseOnEvents on: [pre-commit, push]
func parseOnEvents(unmarshal func(interface{}) error) ([]Listener, error) {
	events := make([]string, 0)
	err := unmarshal(&events)

	// 此时, yaml文件事件有两种可能
	// 2, on:
	//     pre-commit:
	//       branches:
	//         - develop
	//     push:
	//       branches:
	//         - master
	// 第2种情况其实不怎么会出现，暂时不考虑这种格式的解析
	if err != nil {
		return nil, err
	}
	var listeners []Listener
	for _, event := range events {
		listener := Listener{
			Event: event,
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// parseOnEventsWithConditions 此时, yaml文件on事件,类似于:
// on:
//   pre-commit:
//     branches:
//       - develop
func parseOnEventsWithConditions(unmarshal func(interface{}) error) ([]Listener, error) {
	var eventConditions map[string]interface{}
	err := unmarshal(&eventConditions)
	if err != nil {
		return nil, err
	}
	listeners := make([]Listener, 0, len(eventConditions))
	for event, iConditions := range eventConditions {
		listener := Listener{Event: event}
		if conditions, ok := iConditions.(map[string]interface{}); ok {
			listener.Conditions = conditions
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}
