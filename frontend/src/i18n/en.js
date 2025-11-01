export default {
    app: {
        title: 'ccNexus',
        version: 'Version'
    },
    header: {
        title: 'ccNexus - API Endpoint Rotation Proxy',
        port: 'Port',
        addEndpoint: 'Add Endpoint',
        github: 'GitHub'
    },
    endpoints: {
        title: 'Endpoints',
        name: 'Name',
        apiUrl: 'API URL',
        transformer: 'Transformer',
        model: 'Model',
        requests: 'Requests',
        errors: 'Errors',
        tokens: 'Tokens',
        successRate: 'Success Rate',
        actions: 'Actions',
        test: 'Test',
        edit: 'Edit',
        delete: 'Delete',
        noEndpoints: 'No endpoints configured. Click "Add Endpoint" to get started.',
        copy: 'Copy',
        copied: 'Copied'
    },
    modal: {
        addEndpoint: 'Add Endpoint',
        editEndpoint: 'Edit Endpoint',
        name: 'Name',
        apiUrl: 'API URL',
        apiKey: 'API Key',
        transformer: 'Transformer',
        transformerHelp: 'Select the API format for this endpoint',
        model: 'Model',
        modelHelp: 'Optional: Override the model specified in requests',
        modelHelpClaude: 'Optional: Override the model specified in requests',
        modelHelpOpenAI: 'Required: Specify the OpenAI model to use',
        modelHelpGemini: 'Required: Specify the Gemini model to use',
        remark: 'Remark',
        remarkHelp: 'Optional: Add a remark for this endpoint',
        cancel: 'Cancel',
        save: 'Save',
        close: 'Close',
        changePort: 'Change Port',
        port: 'Port',
        portNote: 'Note: Changing port requires application restart'
    },
    logs: {
        title: 'Logs',
        level: 'Level',
        copy: 'Copy',
        clear: 'Clear',
        collapse: 'Collapse',
        expand: 'Expand',
        levels: {
            0: 'DEBUG',
            1: 'INFO',
            2: 'WARN',
            3: 'ERROR'
        }
    },
    test: {
        title: 'Test Result',
        testing: 'Testing...',
        success: 'Success',
        failed: 'Failed'
    },
    welcome: {
        title: 'Welcome to ccNexus!',
        message: 'ccNexus is a smart API endpoint rotation proxy for Claude Code.',
        features: 'Features',
        feature1: 'Automatic failover between multiple API endpoints',
        feature2: 'Support for Claude, OpenAI, and Gemini API formats',
        feature3: 'Real-time statistics and monitoring',
        feature4: 'Smart retry logic with round-robin load balancing',
        getStarted: 'Get Started',
        dontShow: "Don't show this again"
    },
    settings: {
        language: 'Language',
        languages: {
            en: 'English',
            'zh-CN': '简体中文'
        }
    },
    statistics: {
        title: 'Statistics',
        endpoints: 'Endpoints',
        activeTotal: 'Active / Total',
        totalRequests: 'Total Requests',
        success: 'success',
        failed: 'failed',
        totalTokens: 'Total Tokens',
        in: 'In',
        out: 'Out'
    }
};
