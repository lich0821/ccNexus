export default {
    app: {
        title: 'ccNexus',
        version: '版本'
    },
    header: {
        title: 'ccNexus - API 端点轮换代理',
        port: '端口',
        addEndpoint: '添加端点',
        github: 'GitHub'
    },
    endpoints: {
        title: '端点列表',
        name: '名称',
        apiUrl: 'API 地址',
        transformer: '转换器',
        model: '模型',
        requests: '请求数',
        errors: '错误数',
        tokens: 'Token 数',
        successRate: '成功率',
        actions: '操作',
        test: '测试',
        edit: '编辑',
        delete: '删除',
        noEndpoints: '未配置端点。点击"添加端点"开始使用。',
        copy: '复制',
        copied: '已复制'
    },
    modal: {
        addEndpoint: '添加端点',
        editEndpoint: '编辑端点',
        name: '名称',
        apiUrl: 'API 地址',
        apiKey: 'API 密钥',
        transformer: '转换器',
        transformerHelp: '选择此端点的 API 格式',
        model: '模型',
        modelHelp: '可选：覆盖请求中指定的模型',
        modelHelpClaude: '可选：覆盖请求中指定的模型',
        modelHelpOpenAI: '必填：指定要使用的 OpenAI 模型',
        modelHelpGemini: '必填：指定要使用的 Gemini 模型',
        remark: '备注',
        remarkHelp: '可选：为此端点添加备注说明',
        cancel: '取消',
        save: '保存',
        close: '关闭',
        changePort: '修改端口',
        port: '端口',
        portNote: '注意：修改端口需要重启应用'
    },
    logs: {
        title: '日志',
        level: '级别',
        copy: '复制',
        clear: '清空',
        collapse: '收起',
        expand: '展开',
        levels: {
            0: '调试',
            1: '信息',
            2: '警告',
            3: '错误'
        }
    },
    test: {
        title: '测试结果',
        testing: '测试中...',
        success: '成功',
        failed: '失败'
    },
    welcome: {
        title: '欢迎使用 ccNexus！',
        message: 'ccNexus 是一个为 Claude Code 设计的智能 API 端点轮换代理。',
        features: '功能特性',
        feature1: '多个 API 端点之间自动故障转移',
        feature2: '支持 Claude、OpenAI 和 Gemini API 格式',
        feature3: '实时统计和监控',
        feature4: '智能重试逻辑和轮询负载均衡',
        getStarted: '开始使用',
        dontShow: '不再显示'
    },
    settings: {
        language: '语言',
        languages: {
            en: 'English',
            'zh-CN': '简体中文'
        }
    },
    statistics: {
        title: '统计信息',
        endpoints: '端点',
        activeTotal: '活跃 / 总数',
        totalRequests: '总请求数',
        success: '成功',
        failed: '失败',
        totalTokens: '总 Token 数',
        in: '输入',
        out: '输出'
    }
};
