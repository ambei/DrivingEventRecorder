import dayjs from "dayjs";

const flatten = list => list.reduce(
  (a, b) => a.concat(Array.isArray(b) ? flatten(b) : b), []
);

const parseTime = (timeString, date = true, time = true, zone = false) => {
  const template = (date ? 'YYYY-MM-DD' : '') + (date && time ? 'T' : '') + (time ? 'HH:mm:ss' : '') + (zone ? 'Z' : '');
  return dayjs(timeString).format(template);
};

export default {
  flatten, parseTime
}