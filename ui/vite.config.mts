import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react-swc';
// @ts-ignore
import fs from 'fs';

export default defineConfig({
    base: './',
    server: {
        host: '127.0.0.1',
        port: 3000,
        open: false,
        https: {
            key: fs.readFileSync(''),
            cert: fs.readFileSync(''),
        },
        proxy: {
            '^/(config|logout|login|stream)$': {
                target: 'https://127.0.0.1:5050',
                ws: true,
                secure: false
            },
        },
    },
    build: {outDir: 'build/'},
    plugins: [react()],
});
