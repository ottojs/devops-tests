// Modules
import check from 'ssl-checker';
import config from './settings.js';

describe('TLS/SSL Certificate', () => {
	it('should be valid for the next 7 days', async () => {
		const details = await check(config.domain);
		expect(details.valid).toEqual(true);
		expect(details.daysRemaining).toBeGreaterThan(7);
	});
});
