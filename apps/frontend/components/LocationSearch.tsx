"use client";

import { SearchIcon, X } from "lucide-react";
import { InputGroup, InputGroupAddon, InputGroupInput } from "./ui/input-group";
import { Button } from "./ui/button";
import { Spinner } from "./ui/spinner";

interface LocationSearchProps {
  searchTerm: string;
  clearSearch: () => void;
  handleChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onClick: (e: React.MouseEvent<HTMLInputElement>) => void;
  isFetching: boolean;
}

export default function LocationSearch({
  searchTerm,
  clearSearch,
  handleChange,
  onClick,
  isFetching,
}: LocationSearchProps) {
  return (
    <InputGroup size="lg">
      <InputGroupInput
        placeholder="Search locations..."
        value={searchTerm}
        onChange={handleChange}
        onClick={onClick}
      />
      <InputGroupAddon>
        {isFetching ? <Spinner className="size-4" /> : <SearchIcon />}
      </InputGroupAddon>
      {searchTerm.length > 0 && (
        <InputGroupAddon align="inline-end">
          <Button variant="ghost" onClick={clearSearch}>
            <X />
          </Button>
        </InputGroupAddon>
      )}
    </InputGroup>
  );
}
