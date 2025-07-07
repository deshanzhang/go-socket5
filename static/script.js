// 全局变量
let serverStats = {
    connections: 0,
    totalConnections: 0,
    uptime: 0,
    startTime: Date.now(),
    status: 'running'
};

let serverConfig = {
    host: '0.0.0.0',
    port: 1080,
    user: 'test',
    authMethods: ['无认证', '用户名密码认证'],
    maxConnections: 10000,
    blackList: []
};

let activeConnections = [];

// 更新运行时间
function updateUptime() {
    const now = Date.now();
    const uptime = Math.floor((now - serverStats.startTime) / 1000);
    
    const hours = Math.floor(uptime / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    const seconds = uptime % 60;
    
    let uptimeStr = '';
    if (hours > 0) uptimeStr += hours + 'h ';
    if (minutes > 0) uptimeStr += minutes + 'm ';
    uptimeStr += seconds + 's';
    
    const uptimeElement = document.getElementById('uptime');
    if (uptimeElement) {
        uptimeElement.textContent = uptimeStr;
    }
}

// 获取服务器状态
async function fetchServerStats() {
    try {
        const response = await fetch('/api/stats');
        if (response.ok) {
            const stats = await response.json();
            serverStats = { ...serverStats, ...stats };
            updateStatsDisplay();
        }
    } catch (error) {
        console.log('无法获取服务器状态:', error);
        // 使用模拟数据
        serverStats.connections = Math.floor(Math.random() * 10);
        serverStats.totalConnections = Math.floor(Math.random() * 100) + 50;
        updateStatsDisplay();
    }
}

// 获取服务器配置
async function fetchServerConfig() {
    try {
        const response = await fetch('/api/config');
        if (response.ok) {
            const config = await response.json();
            serverConfig = { ...serverConfig, ...config };
            updateConfigDisplay();
        }
    } catch (error) {
        console.log('无法获取服务器配置:', error);
    }
}

// 获取活跃连接
async function fetchActiveConnections() {
    try {
        const response = await fetch('/api/connections');
        if (response.ok) {
            const connections = await response.json();
            activeConnections = connections;
            updateConnectionsDisplay();
        }
    } catch (error) {
        console.log('无法获取活跃连接:', error);
    }
}

// 更新统计显示
function updateStatsDisplay() {
    const connectionsElement = document.getElementById('connections');
    const totalConnectionsElement = document.getElementById('totalConnections');
    const statusElement = document.getElementById('serverStatus');
    
    if (connectionsElement) {
        connectionsElement.textContent = serverStats.connections;
    }
    
    if (totalConnectionsElement) {
        totalConnectionsElement.textContent = serverStats.totalConnections;
    }
    
    if (statusElement) {
        statusElement.textContent = serverStats.status === 'running' ? '✅ 运行中' : '❌ 已停止';
    }
}

// 更新配置显示
function updateConfigDisplay() {
    const configElement = document.getElementById('serverConfig');
    if (configElement) {
        configElement.innerHTML = `
            <p><strong>监听地址:</strong> ${serverConfig.host}:${serverConfig.port}</p>
            <p><strong>认证用户:</strong> ${serverConfig.user}</p>
            <p><strong>最大连接数:</strong> ${serverConfig.maxConnections}</p>
            <p><strong>认证方式:</strong> ${serverConfig.authMethods.join(', ')}</p>
        `;
    }
}

// 更新连接显示
function updateConnectionsDisplay() {
    const connectionsElement = document.getElementById('activeConnections');
    if (connectionsElement) {
        if (activeConnections.length === 0) {
            connectionsElement.innerHTML = '<p>暂无活跃连接</p>';
            return;
        }
        
        let html = '<div class="connections-list">';
        activeConnections.forEach(conn => {
            const duration = Math.floor((Date.now() - new Date(conn.startTime).getTime()) / 1000);
            const durationStr = formatDuration(duration);
            
            html += `
                <div class="connection-item">
                    <div class="connection-info">
                        <strong>${conn.clientIP}</strong> → <strong>${conn.target}</strong>
                        <span class="connection-duration">${durationStr}</span>
                    </div>
                    <button class="button-small" onclick="disconnectConnection('${conn.id}')">断开</button>
                </div>
            `;
        });
        html += '</div>';
        connectionsElement.innerHTML = html;
    }
}

// 格式化持续时间
function formatDuration(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
        return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
        return `${minutes}m ${secs}s`;
    } else {
        return `${secs}s`;
    }
}

// 断开连接
async function disconnectConnection(connectionId) {
    try {
        const response = await fetch(`/api/connections/${connectionId}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('连接已断开', 'success');
            fetchActiveConnections(); // 刷新连接列表
        } else {
            showNotification('断开连接失败', 'error');
        }
    } catch (error) {
        console.log('断开连接失败:', error);
        showNotification('断开连接失败', 'error');
    }
}

// 测试连接
async function testConnection(host, port) {
    try {
        const response = await fetch('/api/test', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ host, port })
        });
        
        if (response.ok) {
            const result = await response.json();
            showNotification(result.message, 'success');
        } else {
            showNotification('连接测试失败', 'error');
        }
    } catch (error) {
        console.log('连接测试失败:', error);
        showNotification('连接测试失败', 'error');
    }
}

// 重启服务器
async function restartServer() {
    if (!confirm('确定要重启服务器吗？')) {
        return;
    }
    
    try {
        const response = await fetch('/api/restart', {
            method: 'POST'
        });
        
        if (response.ok) {
            showNotification('重启请求已发送', 'success');
        } else {
            showNotification('重启请求失败', 'error');
        }
    } catch (error) {
        console.log('重启请求失败:', error);
        showNotification('重启请求失败', 'error');
    }
}

// 刷新状态
function refreshStats() {
    fetchServerStats();
    fetchServerConfig();
    fetchActiveConnections();
    showNotification('状态已刷新', 'success');
}

// 显示通知
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    
    // 添加样式
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        border-radius: 5px;
        color: white;
        font-weight: bold;
        z-index: 1000;
        animation: slideIn 0.3s ease;
        box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    `;
    
    if (type === 'success') {
        notification.style.backgroundColor = '#28a745';
    } else if (type === 'error') {
        notification.style.backgroundColor = '#dc3545';
    } else {
        notification.style.backgroundColor = '#007bff';
    }
    
    document.body.appendChild(notification);
    
    // 3秒后自动移除
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 300);
    }, 3000);
}

// 添加CSS动画
function addNotificationStyles() {
    const style = document.createElement('style');
    style.textContent = `
        @keyframes slideIn {
            from {
                transform: translateX(100%);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
        
        @keyframes slideOut {
            from {
                transform: translateX(0);
                opacity: 1;
            }
            to {
                transform: translateX(100%);
                opacity: 0;
            }
        }
        
        .connections-list {
            max-height: 300px;
            overflow-y: auto;
        }
        
        .connection-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px;
            margin: 5px 0;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 5px;
        }
        
        .connection-info {
            flex: 1;
        }
        
        .connection-duration {
            font-size: 0.8em;
            color: #ccc;
            margin-left: 10px;
        }
        
        .button-small {
            background: #dc3545;
            color: white;
            padding: 5px 10px;
            border: none;
            border-radius: 3px;
            cursor: pointer;
            font-size: 12px;
        }
        
        .button-small:hover {
            background: #c82333;
        }
    `;
    document.head.appendChild(style);
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    addNotificationStyles();
    
    // 初始化数据
    fetchServerStats();
    fetchServerConfig();
    fetchActiveConnections();
    
    // 设置定时器
    setInterval(updateUptime, 1000);
    setInterval(fetchServerStats, 5000);
    setInterval(fetchActiveConnections, 10000);
    
    // 绑定按钮事件
    const refreshButtons = document.querySelectorAll('[onclick="refreshStats()"]');
    refreshButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.preventDefault();
            refreshStats();
        });
    });
    
    console.log('SOCKS5 代理服务器管理面板已加载');
});

// 导出函数供HTML使用
window.refreshStats = refreshStats;
window.showNotification = showNotification;
window.disconnectConnection = disconnectConnection;
window.testConnection = testConnection;
window.restartServer = restartServer; 