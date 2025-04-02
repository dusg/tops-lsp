/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';
import * as vscode from 'vscode';
import * as net from 'net';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
	StreamInfo,
	createServerSocketTransport,
	MessageTransports,
	createClientSocketTransport,
	SocketTransport,
	SocketMessageReader,
	SocketMessageWriter
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
	vscode.window.showInformationMessage('Hello World!');
	// The server is implemented in node
	const serverModule = context.asAbsolutePath(
		path.join('server', 'out', 'server.js')
	);
	const serverOptions: ServerOptions = () => {
		// Create a socket transport
		const port = 4389;
		// return createClientSocketTransport(port).then((socket) => {
		// 	return socket.onConnected().then((transport) => {
		// 		// Do something with the transport
		// 		const reader = transport[0];
		// 		const writer = transport[1];
		// 		// Use the reader and writer as needed
		// 		return {reader: reader, writer: writer};
		// 	})
		// })
		const socket = net.createConnection(port, 'localhost', () => {
			vscode.window.showInformationMessage('Connected to server');
		});
		const reader = new SocketMessageReader(socket);
		const writer = new SocketMessageWriter(socket);
		let transport: MessageTransports = {
			reader: reader,
			writer: writer
		}
		// console.log('Server transport created');
		return Promise.resolve(transport);
	}
	// If the extension is launched in debug mode then the debug server options are used
	// Otherwise the run options are used
	// const serverOptions: ServerOptions = {
	// 	run: { module: serverModule, transport: TransportKind.ipc },
	// 	debug: {
	// 		module: serverModule,
	// 		transport: TransportKind.ipc,
	// 	}
	// };

	// Options to control the language client
	const clientOptions: LanguageClientOptions = {
		// Register the server for plain text documents
		documentSelector: [{ scheme: 'file', language: 'tops' }],
		synchronize: {
			// Notify the server about file changes to '.clientrc files contained in the workspace
			fileEvents: workspace.createFileSystemWatcher('**/.clientrc')
		}
	};

	// Create the language client and start the client.
	client = new LanguageClient(
		'topsLanguageServerExample',
		'Tops Language Server',
		serverOptions,
		clientOptions
	);

	// Start the client. This will also launch the server
	client.start();
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}