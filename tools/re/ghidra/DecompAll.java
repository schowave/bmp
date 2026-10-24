import ghidra.app.script.GhidraScript;
import ghidra.app.decompiler.*;
import ghidra.program.model.listing.*;
import ghidra.program.model.address.*;
import ghidra.app.cmd.function.CreateFunctionCmd;
import java.io.*;
import java.nio.file.*;
import java.util.*;

public class DecompAll extends GhidraScript {
    @Override
    public void run() throws Exception {
        String[] args = getScriptArgs();
        String startsFile = args[0];
        String outFile = args[1];
        Address min = currentProgram.getMinAddress();
        println("min address: " + min);
        SegmentedAddress sm = (SegmentedAddress) min;
        int baseSeg = sm.getSegment();
        for (String line : Files.readAllLines(Paths.get(startsFile))) {
            line = line.trim();
            if (line.isEmpty()) continue;
            String[] p = line.split(":");
            int seg = Integer.parseInt(p[0], 16) + baseSeg;
            int off = Integer.parseInt(p[1], 16);
            Address a = toAddr(String.format("%04x:%04x", seg, off));
            if (getFunctionAt(a) == null) {
                disassemble(a);
                createFunction(a, String.format("f_%04x_%04x", seg - baseSeg, off));
            } else {
                getFunctionAt(a).setName(String.format("f_%04x_%04x", seg - baseSeg, off), ghidra.program.model.symbol.SourceType.USER_DEFINED);
            }
        }
        analyzeChanges(currentProgram);
        DecompInterface di = new DecompInterface();
        di.openProgram(currentProgram);
        PrintWriter pw = new PrintWriter(new FileWriter(outFile));
        FunctionIterator it = currentProgram.getFunctionManager().getFunctions(true);
        int n = 0;
        while (it.hasNext()) {
            Function f = it.next();
            DecompileResults r = di.decompileFunction(f, 120, monitor);
            pw.println("// ==== " + f.getName() + " @ " + f.getEntryPoint());
            if (r != null && r.decompileCompleted()) {
                pw.println(r.getDecompiledFunction().getC());
            } else {
                pw.println("// decompile failed: " + (r == null ? "null" : r.getErrorMessage()));
            }
            n++;
        }
        pw.close();
        println("decompiled " + n);
    }
}
