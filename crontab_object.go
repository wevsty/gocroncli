package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CronExpressionType = int

const (
	EXACT_NUMBER = iota
	RANGE_NUMBER
	MOD_NUMBER
	ANY_NUMBER
)

type CronNumberExpression struct {
	expression_type int
	exact           []int
	min             int
	max             int
	div             int
}

// 自定义类型上实现 UnmarshalJSON 的接口, 在进行 Unmarshal 时就会使用此实现来进行 json 解码
func (self *CronNumberExpression) UnmarshalJSON(bytes_input []byte) error {
	//此除需要去掉传入的数据的两端的 ""
	bytes_value := bytes.Trim(bytes_input, "\"")
	err := self.LoadFromString(string(bytes_value))
	if err != nil {
		// do something
		log.Panic("UnmarshalJSON Failed.")
	}
	return nil
}

// 自定义类型上实现 Marshaler 的接口, 在进行 Marshal 时就会使用此实现来进行 json 编码
func (self *CronNumberExpression) MarshalJSON() ([]byte, error) {
	ret, _ := self.SaveToString()
	return []byte(ret), nil
}

func (self *CronNumberExpression) SaveToString() (string, error) {
	switch {
	case self.expression_type == ANY_NUMBER:
		{
			return "*", nil
		}
	case self.expression_type == RANGE_NUMBER:
		{
			return fmt.Sprintf("%d-%d", self.min, self.max), nil
		}
	case self.expression_type == MOD_NUMBER:
		{
			return fmt.Sprintf("%d/%d", self.min, self.div), nil
		}
	case self.expression_type == EXACT_NUMBER:
		{
			buffer := make([]byte, 0, 10)
			for _, value := range self.exact {
				if len(buffer) != 0 {
					buffer = append(buffer, ',')
				}
				buffer = append(buffer, []byte(strconv.Itoa(value))...)
			}
			return string(buffer), nil
		}
	default:
		{
			return "*", nil
		}
	}
}

func (self *CronNumberExpression) LoadFromString(input string) error {
	switch {
	case input == "*":
		{
			self.expression_type = ANY_NUMBER
		}
	case input == "?":
		{
			self.expression_type = ANY_NUMBER
		}
	case strings.Contains(input, ","):
		{
			// 多个精确的数字
			self.expression_type = EXACT_NUMBER
			string_tuple := strings.Split(input, ",")
			for _, value := range string_tuple {
				converted, err := strconv.Atoi(value)
				if err != nil {
					log.Printf("Parse expression %s error occurred.", input)
					return errors.New("Parse expression error.")
				} else {
					self.exact = append(self.exact, converted)
				}
			}
		}
	case strings.Contains(input, "-"):
		{
			//区间
			//例如：1-2 表示 1到2之间
			self.expression_type = RANGE_NUMBER
			string_tuple := strings.Split(input, ",")
			if len(string_tuple) < 2 {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			converted_min, err := strconv.Atoi(string_tuple[0])
			if err != nil {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			converted_max, err := strconv.Atoi(string_tuple[1])
			if err != nil {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			self.min = converted_min
			self.max = converted_max
		}
	case strings.Contains(input, "/"):
		{
			self.expression_type = MOD_NUMBER
			string_tuple := strings.Split(input, "/")
			if len(string_tuple) < 2 {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			converted_start, err := strconv.Atoi(string_tuple[0])
			if err != nil {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			converted_div, err := strconv.Atoi(string_tuple[1])
			if err != nil {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			self.min = converted_start
			self.div = converted_div
		}
	default:
		{
			//立即数
			converted, err := strconv.Atoi(input)
			if err != nil {
				log.Printf("Parse expression %s error occurred.", input)
				return errors.New("Parse expression error.")
			}
			self.exact = append(self.exact, converted)
		}
	}
	return nil
}

func (self *CronNumberExpression) is_match(input int) bool {
	switch {
	case self.expression_type == ANY_NUMBER:
		{
			return true
		}
	case self.expression_type == RANGE_NUMBER:
		{
			return input >= self.min && input <= self.max
		}
	case self.expression_type == MOD_NUMBER:
		{
			return input >= self.min && (input%self.div == 0)
		}
	case self.expression_type == EXACT_NUMBER:
		{
			for _, value := range self.exact {
				if input == value {
					return true
				}
			}
			return false
		}
	default:
		{
			return false
		}
	}
}

// CronItem Cron配置信息
type CronItem struct {
	Name        string
	StartType   string
	Second      CronNumberExpression
	Minute      CronNumberExpression
	Hour        CronNumberExpression
	Day         CronNumberExpression
	Weekday     CronNumberExpression
	Month       CronNumberExpression
	Year        CronNumberExpression
	Workdir     string
	Exec        string
	Argv        []string
	LastRunTime int64
	ProcessID   int
}

func NewCronItem() *CronItem {
	ptr := new(CronItem)
	ptr.ProcessID = 0
	ptr.LastRunTime = 0
	return ptr
}

func (self *CronItem) LoadCronItemFromJson(json_data []byte) error {
	err := json.Unmarshal(json_data, self)
	if err != nil {
		return err
	}
	return nil
}

func (self *CronItem) is_need_execute(current_time time.Time) bool {
	switch {
	case self.StartType == "ONCE":
		{
			if self.LastRunTime == 0 {
				return true
			}
			return false
		}
	case self.Second.is_match(current_time.Second()) == false:
		{
			return false
		}
	case self.Minute.is_match(current_time.Minute()) == false:
		{
			return false
		}
	case self.Hour.is_match(current_time.Hour()) == false:
		{
			return false
		}
	case self.Day.is_match(current_time.Day()) == false:
		{
			return false
		}
	case self.Weekday.is_match(int(current_time.Weekday())) == false:
		{
			return false
		}
	case self.Month.is_match(int(current_time.Month())) == false:
		{
			return false
		}
	case self.Year.is_match(current_time.Year()) == false:
		{
			return false
		}
	default:
		{
			unix_time := current_time.Unix()
			if self.LastRunTime < unix_time {
				return true
			} else {
				return false
			}
		}
	}
}

func (self *CronItem) execute(current_time time.Time) error {
	if self.is_need_execute(current_time) == true {
		self.LastRunTime = current_time.Unix()
		cmd := exec.Command(self.Exec, self.Argv...)
		cmd.Dir = self.Workdir
		cmd.Env = os.Environ()
		err := cmd.Run()
		if err != nil {
			log.Printf("Start Task [%s] Failed.", self.Name)
			log.Printf(err.Error())
		} else {
			log.Printf("Start Task [%s] PID %d\n", self.Name, cmd.Process.Pid)
			self.ProcessID = cmd.Process.Pid
			//cmd.Process.Release()
		}
	}
	return nil
}
