const test = require('node:test');
const assert = require('node:assert');
const charge = require('./charge');

const amount = { currency_code: 'USD', units: 10, nanos: 0 };

function request(number, year = 2099, month = 12) {
  return {
    amount,
    credit_card: {
      credit_card_number: number,
      credit_card_cvv: 123,
      credit_card_expiration_year: year,
      credit_card_expiration_month: month,
    },
  };
}

test('valid Visa card is charged and returns a transaction id', () => {
  const result = charge(request('4111111111111111'));
  assert.strictEqual(typeof result.transaction_id, 'string');
  assert.ok(result.transaction_id.length > 0);
});

test('valid MasterCard is accepted', () => {
  const result = charge(request('5555555555554444'));
  assert.ok(result.transaction_id);
});

test('invalid card number is rejected', () => {
  assert.throws(
    () => charge(request('1234567890123456')),
    (err) => err.code === 400 && /invalid/i.test(err.message)
  );
});

test('AMEX card is rejected (only VISA and MasterCard accepted)', () => {
  assert.throws(
    () => charge(request('378282246310005')),
    (err) => err.code === 400 && /VISA or MasterCard/.test(err.message)
  );
});

test('expired card is rejected', () => {
  assert.throws(
    () => charge(request('4111111111111111', 2000, 1)),
    (err) => err.code === 400 && /expired/i.test(err.message)
  );
});