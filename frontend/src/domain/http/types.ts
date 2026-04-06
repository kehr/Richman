export interface ApiResponse<T> {
	data: T;
}

export interface PaginatedResponse<T> {
	data: T[];
	pagination: {
		page: number;
		pageSize: number;
		total: number;
		totalPages: number;
	};
}
