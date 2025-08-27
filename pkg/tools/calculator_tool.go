package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CalculatorTool implements mathematical calculations
type CalculatorTool struct{}

// NewCalculatorTool creates a new calculator tool
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// GetName returns the tool name
func (t *CalculatorTool) GetName() string {
	return "calculator"
}

// GetDescription returns the tool description
func (t *CalculatorTool) GetDescription() string {
	return "Perform mathematical calculations and statistical operations"
}

// Execute performs calculations
func (t *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		// Try to evaluate expression
		if expr, ok := args["expression"].(string); ok {
			return t.evaluateExpression(expr)
		}
		return nil, fmt.Errorf("operation or expression parameter is required")
	}
	
	switch operation {
	case "basic":
		return t.basicCalculation(args)
	case "statistics":
		return t.statisticalCalculation(args)
	case "conversion":
		return t.unitConversion(args)
	case "financial":
		return t.financialCalculation(args)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// ValidateArgs validates the arguments
func (t *CalculatorTool) ValidateArgs(args map[string]interface{}) error {
	// Either operation or expression is required
	_, hasOp := args["operation"]
	_, hasExpr := args["expression"]
	
	if !hasOp && !hasExpr {
		return fmt.Errorf("either operation or expression parameter is required")
	}
	
	return nil
}

func (t *CalculatorTool) basicCalculation(args map[string]interface{}) (interface{}, error) {
	operator, ok := args["operator"].(string)
	if !ok {
		return nil, fmt.Errorf("operator parameter is required")
	}
	
	// Get operands
	var a, b float64
	if val, ok := args["a"].(float64); ok {
		a = val
	} else if val, ok := args["a"].(int); ok {
		a = float64(val)
	} else {
		return nil, fmt.Errorf("operand 'a' is required")
	}
	
	if val, ok := args["b"].(float64); ok {
		b = val
	} else if val, ok := args["b"].(int); ok {
		b = float64(val)
	} else if operator != "sqrt" && operator != "abs" {
		return nil, fmt.Errorf("operand 'b' is required for operator %s", operator)
	}
	
	var result float64
	var operation string
	
	switch operator {
	case "+", "add":
		result = a + b
		operation = fmt.Sprintf("%g + %g", a, b)
	case "-", "subtract":
		result = a - b
		operation = fmt.Sprintf("%g - %g", a, b)
	case "*", "multiply":
		result = a * b
		operation = fmt.Sprintf("%g × %g", a, b)
	case "/", "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
		operation = fmt.Sprintf("%g ÷ %g", a, b)
	case "^", "power":
		result = math.Pow(a, b)
		operation = fmt.Sprintf("%g ^ %g", a, b)
	case "sqrt":
		if a < 0 {
			return nil, fmt.Errorf("cannot calculate square root of negative number")
		}
		result = math.Sqrt(a)
		operation = fmt.Sprintf("√%g", a)
	case "abs":
		result = math.Abs(a)
		operation = fmt.Sprintf("|%g|", a)
	default:
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}
	
	return map[string]interface{}{
		"operation": operation,
		"result":    result,
		"formatted": fmt.Sprintf("%s = %g", operation, result),
	}, nil
}

func (t *CalculatorTool) statisticalCalculation(args map[string]interface{}) (interface{}, error) {
	function, ok := args["function"].(string)
	if !ok {
		return nil, fmt.Errorf("function parameter is required")
	}
	
	// Get data array
	var data []float64
	if values, ok := args["data"].([]interface{}); ok {
		for _, v := range values {
			switch val := v.(type) {
			case float64:
				data = append(data, val)
			case int:
				data = append(data, float64(val))
			}
		}
	} else {
		return nil, fmt.Errorf("data array is required")
	}
	
	if len(data) == 0 {
		return nil, fmt.Errorf("data array cannot be empty")
	}
	
	var result interface{}
	
	switch function {
	case "mean", "average":
		sum := 0.0
		for _, v := range data {
			sum += v
		}
		result = sum / float64(len(data))
		
	case "sum":
		sum := 0.0
		for _, v := range data {
			sum += v
		}
		result = sum
		
	case "min":
		min := data[0]
		for _, v := range data[1:] {
			if v < min {
				min = v
			}
		}
		result = min
		
	case "max":
		max := data[0]
		for _, v := range data[1:] {
			if v > max {
				max = v
			}
		}
		result = max
		
	case "count":
		result = len(data)
		
	case "variance":
		mean := 0.0
		for _, v := range data {
			mean += v
		}
		mean /= float64(len(data))
		
		variance := 0.0
		for _, v := range data {
			diff := v - mean
			variance += diff * diff
		}
		result = variance / float64(len(data))
		
	case "stddev":
		mean := 0.0
		for _, v := range data {
			mean += v
		}
		mean /= float64(len(data))
		
		variance := 0.0
		for _, v := range data {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(data))
		result = math.Sqrt(variance)
		
	default:
		return nil, fmt.Errorf("unsupported statistical function: %s", function)
	}
	
	return map[string]interface{}{
		"function": function,
		"data":     data,
		"result":   result,
		"count":    len(data),
	}, nil
}

func (t *CalculatorTool) unitConversion(args map[string]interface{}) (interface{}, error) {
	value, ok := t.getFloat(args["value"])
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}
	
	fromUnit, ok := args["from"].(string)
	if !ok {
		return nil, fmt.Errorf("from unit parameter is required")
	}
	
	toUnit, ok := args["to"].(string)
	if !ok {
		return nil, fmt.Errorf("to unit parameter is required")
	}
	
	// Simple conversion examples
	conversions := map[string]float64{
		"km_m":    1000,
		"m_km":    0.001,
		"kg_g":    1000,
		"g_kg":    0.001,
		"lb_kg":   0.453592,
		"kg_lb":   2.20462,
	}
	
	key := fmt.Sprintf("%s_%s", fromUnit, toUnit)
	var result float64
	
	if fromUnit == "celsius" && toUnit == "fahrenheit" {
		result = (value * 9 / 5) + 32
	} else if fromUnit == "fahrenheit" && toUnit == "celsius" {
		result = (value - 32) * 5 / 9
	} else if factor, ok := conversions[key]; ok {
		result = value * factor
	} else {
		return nil, fmt.Errorf("conversion from %s to %s not supported", fromUnit, toUnit)
	}
	
	return map[string]interface{}{
		"value":     value,
		"from":      fromUnit,
		"to":        toUnit,
		"result":    result,
		"formatted": fmt.Sprintf("%g %s = %g %s", value, fromUnit, result, toUnit),
	}, nil
}

func (t *CalculatorTool) financialCalculation(args map[string]interface{}) (interface{}, error) {
	calcType, ok := args["type"].(string)
	if !ok {
		return nil, fmt.Errorf("calculation type is required")
	}
	
	switch calcType {
	case "compound_interest":
		principal, _ := t.getFloat(args["principal"])
		rate, _ := t.getFloat(args["rate"])
		time, _ := t.getFloat(args["time"])
		n, _ := t.getFloat(args["compounds_per_year"])
		if n == 0 {
			n = 1
		}
		
		// A = P(1 + r/n)^(nt)
		amount := principal * math.Pow(1+rate/(100*n), n*time)
		interest := amount - principal
		
		return map[string]interface{}{
			"principal": principal,
			"rate":      rate,
			"time":      time,
			"amount":    amount,
			"interest":  interest,
		}, nil
		
	case "percentage":
		value, _ := t.getFloat(args["value"])
		percent, _ := t.getFloat(args["percent"])
		
		result := value * percent / 100
		
		return map[string]interface{}{
			"value":   value,
			"percent": percent,
			"result":  result,
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported financial calculation: %s", calcType)
	}
}

func (t *CalculatorTool) evaluateExpression(expr string) (interface{}, error) {
	// Simple expression evaluator (for demo purposes)
	// In production, use a proper expression parser
	expr = strings.TrimSpace(expr)
	
	// Try to parse as number
	if val, err := strconv.ParseFloat(expr, 64); err == nil {
		return map[string]interface{}{
			"expression": expr,
			"result":     val,
		}, nil
	}
	
	// Simple two-operand expressions
	operators := []string{"+", "-", "*", "/", "^"}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) == 2 {
				a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
				b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				
				if err1 == nil && err2 == nil {
					args := map[string]interface{}{
						"operation": "basic",
						"operator":  op,
						"a":         a,
						"b":         b,
					}
					return t.basicCalculation(args)
				}
			}
		}
	}
	
	return nil, fmt.Errorf("unable to evaluate expression: %s", expr)
}

func (t *CalculatorTool) getFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}