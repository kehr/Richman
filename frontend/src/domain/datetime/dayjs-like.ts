// DayjsLike captures the minimal surface of an antd DatePicker value we
// depend on when serialising a form submission. Importing the full dayjs
// type would force adding dayjs as a direct dependency even though antd
// already bundles it transitively, so we keep a structural alias here and
// convert to a native Date for ISO serialisation at the call site.
//
// Usage:
//     const iso = values.tradedAt.toDate().toISOString();
export interface DayjsLike {
	toDate(): Date;
}
