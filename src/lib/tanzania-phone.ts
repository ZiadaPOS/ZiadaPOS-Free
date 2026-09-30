export function tanzaniaPhoneNationalDigits(value: string) {
  let digits = value.replace(/\D/g, '');
  if (digits.startsWith('255')) digits = digits.slice(3);
  if (digits.startsWith('0')) digits = digits.slice(1);
  return digits.slice(0, 9);
}

export function tanzaniaPhoneLocalNumber(value: string) {
  const digits = tanzaniaPhoneNationalDigits(value);
  return digits.length === 9 ? `0${digits}` : '';
}
