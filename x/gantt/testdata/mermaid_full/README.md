# Mermaid Gantt 回归用例

本目录存放 mermaid Gantt 全量语法的正向与反向用例基线，包括综合示例、错误场景与跨平台可复现输入。文件分布：
- `valid_*.gantt`：应成功渲染的输入，搭配测试断言。
- `invalid_*.gantt`：应返回错误的输入，测试行列与错误类型。
- `repro_*.gantt`：用于跨平台/时区一致性验证的输入。
