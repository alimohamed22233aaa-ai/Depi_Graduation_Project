process.env.DISABLE_PROFILER = '1';

const test = require('node:test');
const assert = require('node:assert');
const { getSupportedCurrencies, convert } = require('./server');

test('getSupportedCurrencies returns USD and EUR', (t, done) => {
  getSupportedCurrencies({}, (err, result) => {
    assert.ifError(err);
    assert.ok(result.currency_codes.includes('USD'));
    assert.ok(result.currency_codes.includes('EUR'));
    done();
  });
});

test('convert 100 EUR to EUR returns 100', (t, done) => {
  const call = {
    request: {
      from: { currency_code: 'EUR', units: 100, nanos: 0 },
      to_code: 'EUR',
    },
  };
  convert(call, (err, result) => {
    assert.ifError(err);
    assert.strictEqual(result.units, 100);
    assert.strictEqual(result.currency_code, 'EUR');
    done();
  });
});

test('convert 100 EUR to USD returns 113 units', (t, done) => {
  const call = {
    request: {
      from: { currency_code: 'EUR', units: 100, nanos: 0 },
      to_code: 'USD',
    },
  };
  convert(call, (err, result) => {
    assert.ifError(err);
    assert.strictEqual(result.units, 113);
    assert.strictEqual(result.currency_code, 'USD');
    done();
  });
});