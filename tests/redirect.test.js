// Modules
import req from 'supertest';
import config from './settings.js';

describe('Redirects', () => {
	describe('Root HTTP GET /', () => {
		let res;
		beforeAll(async () => {
			res = await req(`http://${config.domain}`).get('/');
		});
		it('should return status code 301', () => {
			expect(res.statusCode).toEqual(301);
		});
		it('should return redirect HTML', () => {
			expect(res.text).toMatch(/301 Moved Permanently/);
		});
		it('should hide nginx version', () => {
			expect(res.headers.server).toEqual('nginx');
		});
	});
	describe('Root HTTPS GET /', () => {
		let res;
		beforeAll(async () => {
			res = await req(`https://${config.domain}`).get('/');
		});
		it('should return status code 200', () => {
			expect(res.statusCode).toEqual(200);
		});
		it('should return HTML Body', () => {
			expect(res.text).toMatch(/Otto.js/);
		});
		it('should hide nginx version', () => {
			expect(res.headers.server).toEqual('nginx');
		});
	});
	describe('WWW HTTP GET /', () => {
		let res;
		beforeAll(async () => {
			res = await req(`http://www.${config.domain}`).get('/');
		});
		it('should return status code 301', () => {
			expect(res.statusCode).toEqual(301);
		});
		it('should return redirect HTML', () => {
			expect(res.text).toMatch(/301 Moved Permanently/);
		});
		it('should hide nginx version', () => {
			expect(res.headers.server).toEqual('nginx');
		});
	});
	describe('WWW HTTPS GET /', () => {
		let res;
		beforeAll(async () => {
			res = await req(`https://www.${config.domain}`).get('/');
		});
		it('should return status code 301', () => {
			expect(res.statusCode).toEqual(301);
		});
		it('should return redirect HTML', () => {
			expect(res.text).toMatch(/301 Moved Permanently/);
		});
		it('should hide nginx version', () => {
			expect(res.headers.server).toEqual('nginx');
		});
	});
});
