"use client";

import QueryErrorBoundary from "@/components/error/QueryErrorBoundary";
import HostelResults from "@/components/HostelResults";
import HostelResultsError from "@/components/HostelResultsError";
import LocationSearch from "@/components/LocationSearch";
import SearchFilters from "@/components/SearchFilters";
import Footer from "@/components/ui/Footer";
import PropertyCardSkeleton from "@/components/ui/PropertyCardSkeleton";
import { areaFilterParsers, AreaType, hostelFilterParsers } from "@/lib/api/filter";
import { placeSearch } from "@/lib/api/search";
import { APIResponse, Place } from "@/lib/dto";
import { useClickOutside } from "@/lib/hooks/useClickOutside";
import useDebounce from "@/lib/hooks/useDebounce";
import { useQuery } from "@tanstack/react-query";
import { MapPin } from "lucide-react";
import { useQueryStates } from "nuqs";
import { Suspense, useState } from "react";
import { FaMagnifyingGlass } from "react-icons/fa6";

export default function SearchPage() {
  const [hostelFilters, setHostelFilters] = useQueryStates(hostelFilterParsers, {
    history: "push",
  });
  const [areaFilters, setAreaFilters] = useQueryStates(areaFilterParsers, {
    history: "push",
  });
  const [showDropdown, setShowDropdown] = useState(
    areaFilters.searchTerm && areaFilters.searchTerm.length > 0,
  );

  const ref = useClickOutside<HTMLInputElement>(() => setShowDropdown(false));

  const debounceSearchTerm = useDebounce(areaFilters.searchTerm, 300);

  const query = useQuery<APIResponse<Place[]>>({
    queryKey: ["placeSearch", debounceSearchTerm],
    queryFn: ({ signal }) => placeSearch(debounceSearchTerm, { signal }),
    enabled: debounceSearchTerm?.length > 0,
  });

  const clearSearch = () => {
    setAreaFilters({
      searchTerm: null,
      areaType: null,
      areaId: null,
    });
  };

  const handleSearchResultClick = ({
    type,
    id,
    name,
  }: {
    type: AreaType;
    id: string;
    name: string;
  }) => {
    setAreaFilters({
      areaType: type,
      areaId: id,
      searchTerm: name,
    });
    setShowDropdown(false);
  };

  const showDropdownOnClick = (e: React.MouseEvent<HTMLInputElement>) => {
    setShowDropdown(true);
  };

  return (
    <div className="bg-gray-100">
      <hr className="bg-muted-foreground" />
      <main className="pt-12 pb-24 px-8 min-h-screen w-full mx-auto max-w-7xl">
        <div className="flex justify-between items-end">
          <div className="w-full">
            <h1 className="font-headline text-3xl md:text-5xl font-black tracking-tighter mb-10">
              Discover <span className="text-primary">places</span>
            </h1>
            <div className="flex items-center justify-between">
              <div className="relative flex-1 max-w-xl" ref={ref}>
                <LocationSearch
                  searchTerm={areaFilters.searchTerm ? areaFilters.searchTerm : ""}
                  handleChange={e => setAreaFilters({ searchTerm: e.target.value })}
                  clearSearch={clearSearch}
                  onClick={showDropdownOnClick}
                  isFetching={query.isFetching}
                />
                {showDropdown &&
                  areaFilters.searchTerm &&
                  query.isFetched &&
                  (!!query.data?.payload?.length && query.data.payload.length > 0 ? (
                    <div className="absolute z-3 top-full mt-3 left-0 w-full flex flex-col gap-2 bg-primary-foreground shadow-lg ring-1 ring-gray-500/5 p-2 rounded-xl">
                      {query.data.payload.map(searchResult => (
                        <div
                          className="cursor-pointer hover:bg-gray-500/10 p-2 rounded-md flex items-center gap-2 text-muted-foreground"
                          key={searchResult.place_id}
                          onClick={() =>
                            handleSearchResultClick({
                              type: searchResult.place_type as AreaType,
                              id: searchResult.place_id,
                              name: searchResult.name,
                            })
                          }
                        >
                          <MapPin size={12} className="shrink-0" />
                          <span className="text-sm">{searchResult.name}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="absolute z-3 top-full mt-3 left-0 w-full flex flex-col gap-2 bg-primary-foreground shadow-lg ring-1 ring-gray-500/5 p-2 rounded-xl">
                      <span className="p-2 rounded-md flex items-center gap-2 text-muted-foreground text-sm">
                        No matches found
                      </span>
                    </div>
                  ))}
              </div>

              <SearchFilters />
            </div>
          </div>
        </div>
        {areaFilters.areaType &&
        areaFilters.areaId &&
        areaFilters.searchTerm &&
        areaFilters.areaType.length > 0 ? (
          <QueryErrorBoundary errorFallback={HostelResultsError}>
            <Suspense
              fallback={
                <div className="mt-20 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <PropertyCardSkeleton key={i} />
                  ))}
                </div>
              }
            >
              <HostelResults areaName={areaFilters.searchTerm} showInsight />
            </Suspense>
          </QueryErrorBoundary>
        ) : (
          <div className="flex flex-col items-center mt-20 text-muted-foreground">
            <FaMagnifyingGlass size={30} className="mb-4" />
            <h2>Nothing to see here</h2>
            <p>Start typing in the search bar to see results</p>
          </div>
        )}
      </main>
      <Footer />
    </div>
  );
}
