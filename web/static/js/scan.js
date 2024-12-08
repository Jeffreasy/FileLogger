document.addEventListener('alpine:init', () => {
    Alpine.data('scanner', () => ({
        status: 'idle',
        progress: {
            scannedFiles: 0,
            totalFiles: 0,
            blockedFiles: 0
        },
        currentPath: '',
        
        startScan() {
            const config = {
                path: this.$refs.path.value,
                config: {
                    maxFileSizeMB: 50,
                    scanRecursively: true,
                    exportBlockedToJSON: true
                }
            };
            
            fetch('/api/scan', {
                method: 'POST',
                body: JSON.stringify(config)
            });
            
            this.connectWebSocket();
        },
        
        connectWebSocket() {
            const ws = new WebSocket('ws://localhost:8080/api/ws');
            
            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.progress = data.progress;
                this.currentPath = data.currentDirectory;
            };
        }
    }));
}); 