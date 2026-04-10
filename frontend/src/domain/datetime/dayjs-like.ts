// DayjsLike captures the minimal surface of an antd DatePicker / TimePicker
// value we depend on when serialising a form submission or reading a time.
// Importing the full dayjs type would force adding dayjs as a direct
// dependency even though antd already bundles it transitively, so we keep a
// structural alias here and convert at the call site.
//
// Usage (DatePicker):
//     const iso = values.tradedAt.toDate().toISOString();
//
// Usage (TimePicker):
//     const hhmm = values.time.format("HH:mm");
export interface DayjsLike {
	toDate(): Date;
	format(fmt: string): string;
	hour(): number;
	minute(): number;
}
