# xssh

[English](README.md)

在一个终端窗口中同时连接多台服务器。支持并排显示、广播命令、滚动历史记录，一切尽在掌握。

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/平台-macOS%20%7C%20Linux-lightgrey" alt="平台">
  <img src="https://img.shields.io/badge/许可证-MIT-blue" alt="许可证">
</p>

## 功能特性

- **最多 9 个窗格** — SSH 会话和本地 Shell 自动排列成网格（1×1 → 3×3）
- **广播输入** — 实时按键转发到多个窗格，支持选择性发送
- **滚动与搜索** — 保留 ANSI 颜色的滚动缓冲区，支持搜索
- **读取 SSH 配置** — 模糊搜索 `~/.ssh/config` 中的主机别名
- **会话分组** — 保存和加载多主机配置方案
- **共享边框** — 相邻窗格共用单线边框，节省空间（可切换）
- **自动重连** — SSH 连接断开后自动重试
- **鼠标支持** — 点击切换焦点，滚轮浏览历史
- **缩放** — 将任意窗格全屏显示，随时恢复网格

## 安装

```bash
go install github.com/xssh/xssh@latest
```

或从源码构建：

```bash
git clone https://github.com/xssh/xssh.git
cd xssh
go build -o xssh .
```

## 快速开始

```bash
# 交互式主机选择器（读取 ~/.ssh/config）
xssh

# 两个本地 Shell 并排显示
xssh - -

# 连接三台服务器
xssh web1 web2 db1

# 混合本地和远程
xssh - user@192.168.1.10 staging

# 加载保存的分组
xssh -g production
```

## 快捷键

所有快捷键以 **Ctrl+\\** 为前缀键，然后按功能键。

| 按键 | 功能 |
|------|------|
| `Ctrl+\ 1-9` | 切换到窗格 1-9 |
| `Ctrl+\ h/j/k/l` | 焦点左移/下移/上移/右移 |
| `Ctrl+\ z` | 缩放/还原当前窗格 |
| `Ctrl+\ x` | 关闭当前窗格 |
| `Ctrl+\ r` | 重连当前窗格 |
| `Ctrl+\ R` | 重连所有窗格 |
| `Ctrl+\ b` | 实时广播输入（切换） |
| `Ctrl+\ m` | 选择接收广播的窗格 |
| `Ctrl+\ [` | 进入滚动模式 |
| `Ctrl+\ e` | 添加新窗格 |
| `Ctrl+\ s` | 保存当前会话为分组 |
| `Ctrl+\ ?` | 显示帮助 |
| `Ctrl+\ \` | 向会话发送 Ctrl+\ |

### 滚动模式

| 按键 | 功能 |
|------|------|
| `↑/k` `↓/j` | 向上/向下滚动 |
| `PgUp` `PgDn` | 半页滚动 |
| `g` / `G` | 跳转到顶部/底部 |
| `/` | 搜索 |
| `n` / `N` | 下一个/上一个匹配 |
| `q` `Esc` | 退出滚动模式 |

## 命令行参数

```
xssh [参数] [目标...]

目标:
  -                     本地 Shell
  user@host             SSH 连接
  alias                 SSH 配置别名

参数:
  -h, --help            显示帮助
  -g, --group 名称      加载保存的分组
  --save 名称 目标…     将目标保存为分组
  --list-groups         列出所有分组
  --list-hosts          列出 SSH 配置中的主机
  --borders 模式        边框样式: shared（默认）或 full
```

## 边框模式

**共享边框**（默认 — `--borders shared`）：
```
╭──────┬──────╮
│ 窗格1 │ 窗格2 │
├──────┼──────┤
│ 窗格3 │ 窗格4 │
├──────┴──────┤
│ 输入栏       │
╰──────────────╯
```

**独立边框**（`--borders full`）：
```
╭──────╮╭──────╮
│ 窗格1 ││ 窗格2 │
╰──────╯╰──────╯
╭──────╮╭──────╮
│ 窗格3 ││ 窗格4 │
╰──────╯╰──────╯
╭──────────────╮
│ 输入栏       │
╰──────────────╯
```

## 配置文件

配置路径：`~/.xssh/config.yaml`

```yaml
general:
  scrollback_lines: 5000      # 滚动缓冲区行数
  reconnect_attempts: 3       # 重连尝试次数
  reconnect_interval: 5s      # 重连间隔
  ssh_timeout: 10s            # SSH 连接超时

groups:
  production:
    - web1
    - web2
    - db-master
  staging:
    - staging-web
    - staging-db
```

## 布局网格

| 窗格数 | 网格 |
|--------|------|
| 1 | 1×1 |
| 2 | 1×2 |
| 3-4 | 2×2 |
| 5-6 | 3×2 |
| 7-9 | 3×3 |

## 系统要求

- Go 1.25+
- 支持 256 色和鼠标的终端
- macOS 或 Linux

## 许可证

MIT
