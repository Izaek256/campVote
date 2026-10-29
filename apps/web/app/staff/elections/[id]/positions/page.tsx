"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { Position } from "@campvote/shared/types/position";

export default function PositionsPage() {
  const params = useParams();
  const electionId = params.id;
  const [positions, setPositions] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    fetchPositions();
  }, [electionId]);

  const fetchPositions = async () => {
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/positions`);
      if (!response.ok) throw new Error("Failed to fetch positions");
      const data = await response.json();
      setPositions(data.positions || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;

  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Positions</h1>
        <button
          onClick={() => setShowForm(true)}
          className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
        >
          Add Position
        </button>
      </div>

      {positions.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          No positions yet. Add your first position to get started.
        </div>
      ) : (
        <div className="space-y-4">
          {positions.map((position) => (
            <div key={position.id} className="border rounded-lg p-4">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-semibold">{position.title}</h3>
                  {position.description && (
                    <p className="text-gray-600 text-sm mt-1">{position.description}</p>
                  )}
                  <div className="mt-2 text-sm text-gray-500">
                    <span>Type: {position.question_type}</span>
                    <span className="ml-4">Seats: {position.seats}</span>
                    <span className="ml-4">
                      Selections: {position.min_selections}-{position.max_selections}
                    </span>
                  </div>
                </div>
                <div className="flex gap-2">
                  <button className="px-3 py-1 text-sm border rounded hover:bg-gray-50">
                    Edit
                  </button>
                  <button className="px-3 py-1 text-sm text-red-600 border border-red-200 rounded hover:bg-red-50">
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Add Position</h2>
            <form className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Title</label>
                <input
                  type="text"
                  className="w-full border rounded px-3 py-2"
                  placeholder="e.g., Guild President"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Description</label>
                <textarea
                  className="w-full border rounded px-3 py-2"
                  rows={3}
                  placeholder="Position description..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Question Type</label>
                <select className="w-full border rounded px-3 py-2">
                  <option value="SINGLE_CHOICE">Single Choice</option>
                  <option value="MULTI_CHOICE">Multi Choice</option>
                  <option value="YES_NO">Yes/No</option>
                  <option value="REFERENDUM">Referendum</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Seats</label>
                  <input
                    type="number"
                    min="1"
                    defaultValue="1"
                    className="w-full border rounded px-3 py-2"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Max Selections</label>
                  <input
                    type="number"
                    min="1"
                    defaultValue="1"
                    className="w-full border rounded px-3 py-2"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <button
                  type="button"
                  onClick={() => setShowForm(false)}
                  className="px-4 py-2 border rounded hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                >
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
