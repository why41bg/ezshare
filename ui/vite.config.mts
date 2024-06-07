import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react-swc';

export default defineConfig({
    base: './',
    server: {
        host: '192.168.0.104',
        port: 3000,
        open: false,
        proxy: {
            '^/(config|logout|login|stream)$': {
                target: 'https://localhost:5050',
                ws: true,
                secure: false, // 忽略证书验证
            },
        },
    },
    build: {outDir: 'build/'},
    plugins: [react()],
});
