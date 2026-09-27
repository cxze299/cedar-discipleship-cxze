import { describe, expect, it, vi } from 'vitest';

import { nextReadingStartPage, saveWeekWithConfirmation } from './weekProtection';

describe('saveWeekWithConfirmation', () => {
  it('saves once when no check-in conflict exists', async () => {
    const send = vi.fn().mockResolvedValue({ id: 7 });
    const confirm = vi.fn();

    await expect(saveWeekWithConfirmation(send, confirm)).resolves.toEqual({ id: 7 });
    expect(send).toHaveBeenCalledTimes(1);
    expect(send).toHaveBeenCalledWith(false);
    expect(confirm).not.toHaveBeenCalled();
  });

  it('does not retry when force modification is cancelled', async () => {
    const error = Object.assign(new Error('week_has_checkins'), { code: 'week_has_checkins' });
    const send = vi.fn().mockRejectedValue(error);
    const confirm = vi.fn().mockReturnValue(false);

    await expect(saveWeekWithConfirmation(send, confirm)).resolves.toBeNull();
    expect(send).toHaveBeenCalledTimes(1);
    expect(confirm).toHaveBeenCalledTimes(1);
  });

  it('retries exactly once with force after confirmation', async () => {
    const error = Object.assign(new Error('week_has_checkins'), { code: 'week_has_checkins' });
    const send = vi.fn()
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce({ id: 7 });
    const confirm = vi.fn().mockReturnValue(true);

    await expect(saveWeekWithConfirmation(send, confirm)).resolves.toEqual({ id: 7 });
    expect(send.mock.calls).toEqual([[false], [true]]);
  });

  it('propagates unrelated errors without confirmation', async () => {
    const error = Object.assign(new Error('week_task_save_failed'), { code: 'week_task_save_failed' });
    const send = vi.fn().mockRejectedValue(error);
    const confirm = vi.fn();

    await expect(saveWeekWithConfirmation(send, confirm)).rejects.toBe(error);
    expect(confirm).not.toHaveBeenCalled();
  });

  it.each([false, true])('waits for the dialog before acting on %s', async (confirmed) => {
    let resolveConfirmation!: (value: boolean) => void;
    const decision = new Promise<boolean>((resolve) => { resolveConfirmation = resolve; });
    const send = vi.fn()
      .mockRejectedValueOnce(Object.assign(new Error(), { code: 'week_has_checkins' }))
      .mockResolvedValue({ id: 7 });
    const confirm = vi.fn(() => decision);
    const pendingSave = saveWeekWithConfirmation(send, confirm);
    await vi.waitFor(() => expect(confirm).toHaveBeenCalledOnce());
    const callsBeforeDecision = [...send.mock.calls];
    resolveConfirmation(confirmed);
    await expect(pendingSave).resolves.toEqual(confirmed ? { id: 7 } : null);
    expect(callsBeforeDecision).toEqual([[false]]);
    expect(send.mock.calls).toEqual(confirmed ? [[false], [true]] : [[false]]);
  });

  it('propagates a failed forced save without prompting again', async () => {
    const failure = new Error('save failed');
    const send = vi.fn()
      .mockRejectedValueOnce(Object.assign(new Error(), { code: 'week_has_checkins' }))
      .mockRejectedValueOnce(failure);
    const confirm = vi.fn().mockResolvedValue(true);
    await expect(saveWeekWithConfirmation(send, confirm)).rejects.toBe(failure);
    expect(confirm).toHaveBeenCalledOnce();
    expect(send.mock.calls).toEqual([[false], [true]]);
  });
});

describe('nextReadingStartPage', () => {
  it('reuses the previous ending page for the next week', () => {
    expect(nextReadingStartPage('22')).toBe('22');
  });

  it('keeps the next start empty when the previous range has no ending page', () => {
    expect(nextReadingStartPage('')).toBe('');
  });
});
