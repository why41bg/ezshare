import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react-swc';
// @ts-ignore
import fs from 'fs';

export default defineConfig({
    base: './',
    server: {
        host: '192.168.0.104',
        port: 3000,
        open: false,
        https: {
            key: fs.readFileSync('D:\\Project\\Project\\ezshare\\private.key'),
            cert: fs.readFileSync('D:\\Project\\Project\\ezshare\\crt.pem'),
        },
        proxy: {
            '^/(config|logout|login|stream)$': {
                target: 'https://192.168.0.104:5050',
                ws: true,
                secure: false
            },
        },
    },
    build: {outDir: 'build/'},
    plugins: [react()],
});
