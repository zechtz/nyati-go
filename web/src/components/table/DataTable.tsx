import React, { ReactNode } from "react";
import {
  Table as UITable,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";

export interface Column {
  key: string;
  label: string;
  width?: string;
  align?: "left" | "center" | "right";
  sortable?: boolean;
  render?: (value: any, row: any) => ReactNode;
}

export interface PaginationParams {
  page: number;
  perPage: number;
  total: number;
}

export interface TableProps {
  columns: Column[];
  items: any[];
  size?: "default" | "small" | "large";
  showPagination?: boolean;
  params?: PaginationParams;
  onPagination?: (params: PaginationParams) => void;
  onSort?: (key: string, direction: "asc" | "desc") => void;
  children?: (props: { row: any; index: number }) => ReactNode;
}

interface SlotProps {
  name: string;
  children: ReactNode;
}

// Slot component to be used inside the render prop
export const Slot: React.FC<SlotProps> = ({ children }) => {
  return <>{children}</>;
};

export const DataTable: React.FC<TableProps> = ({
  columns,
  items = [],
  size = "default",
  showPagination = false,
  params = { page: 1, perPage: 10, total: 0 },
  onPagination,
  children,
}) => {
  // Handle pagination change
  const handlePageChange = (newPage: number) => {
    if (onPagination) {
      onPagination({
        ...params,
        page: newPage,
      });
    }
  };

  // Calculate pagination values
  const totalPages = Math.ceil(params.total / params.perPage);
  const startItem = (params.page - 1) * params.perPage + 1;
  const endItem = Math.min(params.page * params.perPage, params.total);

  // Get cell size based on the size prop
  const getCellClass = () => {
    switch (size) {
      case "small":
        return "py-2 px-3";
      case "large":
        return "py-4 px-6";
      default:
        return "py-3 px-4";
    }
  };

  // Process row cells with slots
  const processRow = (row: any, index: number) => {
    // If no children provided, just render the cells normally
    if (!children) {
      return columns.map((column) => (
        <TableCell key={column.key} className={getCellClass()}>
          {column.render
            ? column.render(row[column.key], row)
            : row[column.key]}
        </TableCell>
      ));
    }

    // Otherwise, process the children to find slots
    const rendered = children({ row, index });

    if (!React.isValidElement(rendered)) {
      return columns.map((column) => (
        <TableCell key={column.key} className={getCellClass()}>
          {column.render
            ? column.render(row[column.key], row)
            : row[column.key]}
        </TableCell>
      ));
    }

    // Get the slots from the children
    const slots: Record<string, ReactNode> = {};
    React.Children.forEach(rendered as React.ReactElement, (child) => {
      if (React.isValidElement(child) && child.type === Slot) {
        const slotName = (child.props as SlotProps).name;
        slots[slotName] = (child.props as SlotProps).children;
      }
    });

    // Render cells with slots
    return columns.map((column) => {
      // Check if there's a slot for this column
      if (slots[column.key]) {
        return (
          <TableCell key={column.key} className={getCellClass()}>
            {slots[column.key]}
          </TableCell>
        );
      }

      // For actions column, use the actions slot if available
      if (column.key === "actions" && slots.actions) {
        return (
          <TableCell key="actions" className={getCellClass()}>
            {slots.actions}
          </TableCell>
        );
      }

      // Default rendering
      return (
        <TableCell key={column.key} className={getCellClass()}>
          {column.render
            ? column.render(row[column.key], row)
            : row[column.key]}
        </TableCell>
      );
    });
  };

  return (
    <div className="w-full">
      <div className="rounded-md border">
        <UITable>
          <TableHeader>
            <TableRow>
              {columns.map((column) => (
                <TableHead
                  key={column.key}
                  style={{ width: column.width }}
                  className={column.align ? `text-${column.align}` : ""}
                >
                  {column.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="text-center py-6"
                >
                  <p>No data available</p>
                </TableCell>
              </TableRow>
            ) : (
              items.map((row, index) => (
                <TableRow key={index}>{processRow(row, index)}</TableRow>
              ))
            )}
          </TableBody>
        </UITable>
      </div>

      {showPagination && params.total > 0 && (
        <div className="flex items-center justify-between space-x-2 py-4">
          <div className="text-sm text-muted-foreground">
            Showing {startItem} to {endItem} of {params.total} entries
          </div>
          <div className="flex space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(1)}
              disabled={params.page === 1}
            >
              <ChevronsLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(params.page - 1)}
              disabled={params.page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <div className="flex items-center text-sm">
              Page {params.page} of {totalPages}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(params.page + 1)}
              disabled={params.page === totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(totalPages)}
              disabled={params.page === totalPages}
            >
              <ChevronsRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default DataTable;
