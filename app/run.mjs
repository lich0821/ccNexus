#!/usr/bin/env node

/**
 * ccNexus 运行脚本
 * 默认: 开发模式
 * 构建: node run.mjs -b 或 node run.mjs --build
 */

import { spawn, exec } from 'child_process'
import { promisify } from 'util'
import { existsSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const execAsync = promisify(exec)
const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

// 颜色输出
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
}

const log = {
  info: (msg) => console.log(`${colors.blue}ℹ${colors.reset} ${msg}`),
  success: (msg) => console.log(`${colors.green}✓${colors.reset} ${msg}`),
  error: (msg) => console.log(`${colors.red}✗${colors.reset} ${msg}`),
  warn: (msg) => console.log(`${colors.yellow}⚠${colors.reset} ${msg}`),
  title: (msg) => console.log(`\n${colors.bright}${colors.cyan}${msg}${colors.reset}\n`),
}

// 检查命令是否存在
async function commandExists(cmd) {
  try {
    const command = process.platform === 'win32' ? `where ${cmd}` : `which ${cmd}`
    await execAsync(command)
    return true
  } catch {
    return false
  }
}

// 执行命令并实时输出
function runCommand(cmd, args = [], options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, {
      stdio: 'inherit',
      shell: true,
      ...options,
    })

    child.on('close', (code) => {
      if (code === 0) {
        resolve()
      } else {
        reject(new Error(`命令执行失败，退出码: ${code}`))
      }
    })

    child.on('error', reject)
  })
}

// 检查前端依赖
function checkFrontendDeps() {
  const nodeModulesPath = join(__dirname, 'frontend', 'node_modules')
  return existsSync(nodeModulesPath)
}

// 安装前端依赖
async function installFrontendDeps() {
  log.info('📦 安装前端依赖...')
  const frontendDir = join(__dirname, 'frontend')

  // 检测是否在国内网络环境
  const useRegistry = process.env.NPM_CONFIG_REGISTRY || 'https://registry.npmmirror.com'
  log.info(`使用 NPM 镜像: ${useRegistry}`)

  try {
    await runCommand('npm', ['install', '--registry', useRegistry], { cwd: frontendDir })
    log.success('前端依赖安装完成')
  } catch (error) {
    log.error('前端依赖安装失败')
    throw error
  }
}

// 安装 Wails
async function installWails() {
  log.info('🔧 准备安装 Wails...')

  // 检查 Go 是否安装
  if (!(await commandExists('go'))) {
    log.error('未找到 Go 命令，请先安装 Go: https://golang.org/dl/')
    process.exit(1)
  }

  // 配置国内镜像
  const goEnv = {
    ...process.env,
    GOPROXY: 'https://goproxy.cn,direct',
    GOSUMDB: 'sum.golang.org',
  }

  log.info('📦 使用国内镜像加速安装...')
  log.info('GOPROXY=https://goproxy.cn,direct')

  try {
    // 安装 Wails
    await runCommand(
      'go',
      ['install', 'github.com/wailsapp/wails/v2/cmd/wails@latest'],
      { env: goEnv }
    )
    log.success('Wails 安装成功！')

    // 提示添加 GOPATH/bin 到 PATH
    log.warn('请确保 $GOPATH/bin 或 $HOME/go/bin 已添加到 PATH 环境变量')
  } catch (error) {
    log.error('Wails 安装失败')
    throw error
  }
}

// 检查 Wails
async function checkWails() {
  if (!(await commandExists('wails'))) {
    log.warn('未找到 wails 命令')
    log.info('正在自动安装 Wails (使用国内镜像加速)...')

    try {
      await installWails()

      // 再次检查
      if (!(await commandExists('wails'))) {
        log.error('Wails 安装后仍未找到命令')
        log.info('请手动添加 $GOPATH/bin 到 PATH，或重启终端后再试')
        log.info('GOPATH 路径: ' + (process.env.GOPATH || '$HOME/go'))
        process.exit(1)
      }
    } catch (error) {
      log.error('自动安装失败，请手动安装:')
      log.info('GOPROXY=https://goproxy.cn,direct go install github.com/wailsapp/wails/v2/cmd/wails@latest')
      process.exit(1)
    }
  }
}

// 开发模式
async function dev() {
  log.title('🚀 启动 ccNexus 开发模式')

  await checkWails()

  // 检查前端依赖
  if (!checkFrontendDeps()) {
    await installFrontendDeps()
  }

  // 启动开发服务器
  log.info('🔧 启动开发服务器...')
  try {
    await runCommand('wails', ['dev'])
  } catch (error) {
    log.error('开发服务器启动失败')
    process.exit(1)
  }
}

// 构建
async function build(options = {}) {
  log.title('🏗️  构建 ccNexus')

  await checkWails()

  // 检查前端依赖
  if (!checkFrontendDeps()) {
    await installFrontendDeps()
  }

  // 构建参数
  const args = ['build']

  if (options.clean !== false) {
    args.push('-clean')
  }

  if (options.prod) {
    args.push('-upx', '-ldflags', '-w -s')
    log.info('🎯 生产模式构建（启用优化和压缩）')
  }

  if (options.platform) {
    args.push('-platform', options.platform)
    log.info(`🎯 构建平台: ${options.platform}`)
  }

  // 执行构建
  log.info(`执行: wails ${args.join(' ')}`)
  try {
    await runCommand('wails', args)
    log.success('✅ 构建完成！输出位置: build/bin/')
  } catch (error) {
    log.error('构建失败')
    process.exit(1)
  }
}

// 显示帮助信息
function showHelp() {
  console.log(`
${colors.bright}${colors.cyan}ccNexus 运行脚本${colors.reset}

${colors.bright}用法:${colors.reset}
  node run.mjs [选项]

${colors.bright}选项:${colors.reset}
  ${colors.green}无参数${colors.reset}              开发模式（默认）
  ${colors.green}-b, --build${colors.reset}        构建模式
  ${colors.green}-p, --prod${colors.reset}         生产构建（优化+压缩）
  ${colors.green}--platform <平台>${colors.reset}   指定构建平台
  ${colors.green}-h, --help${colors.reset}         显示帮助信息

${colors.bright}平台选项:${colors.reset}
  windows/amd64        Windows 64位
  darwin/universal     macOS 通用版本
  darwin/amd64         macOS Intel
  darwin/arm64         macOS Apple Silicon
  linux/amd64          Linux 64位

${colors.bright}示例:${colors.reset}
  ${colors.cyan}node run.mjs${colors.reset}                    # 开发模式
  ${colors.cyan}node run.mjs -b${colors.reset}                 # 标准构建
  ${colors.cyan}node run.mjs --build --prod${colors.reset}     # 生产构建
  ${colors.cyan}node run.mjs -b --platform windows/amd64${colors.reset}  # 构建 Windows 版本

${colors.bright}快捷方式:${colors.reset}
  ${colors.cyan}npm run dev${colors.reset}      或  ${colors.cyan}./dev.sh${colors.reset}     # 开发模式
  ${colors.cyan}npm run build${colors.reset}    或  ${colors.cyan}./build.sh${colors.reset}   # 构建模式
`)
}

// 主函数
async function main() {
  const args = process.argv.slice(2)

  // 显示帮助
  if (args.includes('-h') || args.includes('--help')) {
    showHelp()
    return
  }

  // 判断是构建还是开发
  const isBuild = args.includes('-b') || args.includes('--build')
  const isProd = args.includes('-p') || args.includes('--prod')

  // 获取平台参数
  let platform = null
  const platformIndex = args.indexOf('--platform')
  if (platformIndex !== -1 && args[platformIndex + 1]) {
    platform = args[platformIndex + 1]
  }

  try {
    if (isBuild) {
      await build({
        prod: isProd,
        platform
      })
    } else {
      await dev()
    }
  } catch (error) {
    log.error(error.message)
    process.exit(1)
  }
}

// 执行
main()
