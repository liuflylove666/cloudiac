export const toPercent = (value) => {
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return 0;
  }
  return Math.round(number * 10000) / 100;
};

export const ratioToPercent = (value, total) => {
  const denominator = Number(total);
  if (!Number.isFinite(denominator) || denominator <= 0) {
    return 0;
  }
  return toPercent(Number(value || 0) / denominator);
};

export const numberOrZero = (value) => {
  const number = Number(value);
  return Number.isFinite(number) ? number : 0;
};
