import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';
import * as vscode from 'vscode';
import * as net from 'net';
import * as child_process from 'child_process';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	MessageTransports,
	SocketMessageReader,
	SocketMessageWriter
} from 'vscode-languageclient/node';

let client: LanguageClient;
let serverProcess: child_process.ChildProcess | undefined;
let outputChannel: vscode.OutputChannel;

export function activate(context: ExtensionContext) {
	outputChannel = vscode.window.createOutputChannel('Tops LSP');
	outputChannel.appendLine('Tops LSP Output initialized.');
	outputChannel.show();

	// 注册重启 LSP 服务器的命令
	context.subscriptions.push(
		vscode.commands.registerCommand('tops-lsp.restartServer', () => {
			restartServer();
		})
	);

	startServer();
}

function restartServer() {
	outputChannel.appendLine('LSP Server is restarting...');
	if (client) {
		client.stop().then(() => {
			startServer(); // 重新启动服务器
		});
	} else {
		startServer(); // 如果客户端未启动，直接启动服务器
	}
}

function createLanguageClient(serverAddress: string): LanguageClient {
	const serverOptions: ServerOptions = () => {
		if (!serverAddress) {
			return Promise.reject(new Error('Server address not available'));
		}
		const [host, port] = serverAddress.split(':');
		const socket = net.createConnection(Number(port), host, () => {
			vscode.window.showInformationMessage('Connected to server');
		});
		const reader = new SocketMessageReader(socket);
		const writer = new SocketMessageWriter(socket);
		let transport: MessageTransports = {
			reader: reader,
			writer: writer
		};
		return Promise.resolve(transport);
	};

	// Options to control the language client
	const clientOptions: LanguageClientOptions = {
		// Register the server for plain text documents
		documentSelector: [{ scheme: 'file', language: 'tops' }],
		synchronize: {
			// Notify the server about file changes to '.clientrc files contained in the workspace
			fileEvents: workspace.createFileSystemWatcher('**/.clientrc')
		}
	};

	// Create the language client and return it
	return new LanguageClient(
		'enflame.carl_du.tops.lsp',
		'Tops Language Server',
		serverOptions,
		clientOptions
	);
}

function startServer() {
	const workspacePath = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
	if (!workspacePath) {
		vscode.window.showErrorMessage('No workspace folder is open.');
		return;
	}

	serverProcess = child_process.spawn(
		vscode.workspace.getConfiguration('tops-lsp').get<string>('serverPath') ||
		path.join('bin', 'tops-lsp'),
		[`--ws`, workspacePath], // 传入 workspace 参数
		{ shell: true }
	);
	let serverAddress: string | undefined;

	serverProcess.stdout.on('data', (data) => {
		const output = data.toString();
		outputChannel.appendLine(output); // 将输出写入 Output View
		const match = output.match(/running on (.+)/);
		if (match) {
			serverAddress = match[1].trim();

			// 使用提取的函数创建并启动 LanguageClient
			client = createLanguageClient(serverAddress);
			client.start();
		}
	});

	serverProcess.stderr.on('data', (data) => {
		const errorOutput = data.toString();
		outputChannel.appendLine(errorOutput); // 将错误输出写入 Output View
		// vscode.window.showErrorMessage(`LSP Server error: ${errorOutput}`);
	});

	serverProcess.on('exit', (code) => {
		outputChannel.appendLine(`LSP Server exited with code ${code}`); // 将退出信息写入 Output View
		vscode.window.showInformationMessage(`LSP Server exited with code ${code}`);
	});
}

export function deactivate(): Thenable<void> | undefined {
	if (outputChannel) {
		outputChannel.dispose(); // 释放 Output View
	}
	if (!client) {
		return undefined;
	}
	if (!serverProcess) {
		serverProcess.kill();
	}
	return client.stop();
}